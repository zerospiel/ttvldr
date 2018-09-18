package downloader

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	twitchClient           = "o4m8ilgpeewree25zlyzr1noba1j7t"
	defaultQuality         = "chunked"
	tsExtension            = ".ts"
	regTSfile              = ".+[.]ts"
	regQualityAndM3U8List  = "(VIDEO=\\\".*)\\n.*(\\.m3u8)"
	regDuration            = "#EXTINF:(\\d+)[.].+"
	targetDurationStrBegin = "TARGETDURATION:"
	targetDurationStrEnd   = "\n#ID3"
	newAPIGetVideo         = "https://api.twitch.tv/helix/videos?id="
	oldAPIGetVideo         = "https://api.twitch.tv/api/vods/%VODIDREPLACER%/access_token?&client_id="
	ffmpegBinary           = "ffmpeg"
	goroutinsLimit         = 8
)

var (
	sem = make(chan struct{}, goroutinsLimit)
	// Debug is a flag that enables debug prints
	Debug bool
	// TimeF is a flag that enables time prints
	TimeF bool
)

func init() {
	for i := 0; i < goroutinsLimit; i++ {
		sem <- struct{}{}
	}
	f := false
	cmd := exec.Command(ffmpegBinary)
	if b, _ := cmd.Output(); b != nil {
		f = true
	}
	if !f {
		ext := ""
		dir, _ := filepath.Abs(".")
		if runtime.GOOS == "windows" {
			ext += ".exe"
		}
		fmt.Printf("There is no ffmpeg%s in current directory %s.\n\nYou can download ffmpeg following this link: %s.\n\nPlace ffmpeg%s in directory with 'ttvldr'.", ext, dir, "https://www.ffmpeg.org/download.html", ext)
		os.Exit(1)
	}
}

func getToken(vodID string) (token string, sig string, err error) {
	twitchAPIv2 := strings.Replace(oldAPIGetVideo, "%VODIDREPLACER%", vodID, 1)
	twitchAPIv2 += twitchClient
	debugPrintf("\nLink to v2 API: %s\n", twitchAPIv2)
	resp, err := http.Get(twitchAPIv2)
	if err != nil {
		return "", "", fmt.Errorf("getToken: cannot get twitch API v2 token. %s", err.Error())
	}
	defer resp.Body.Close()

	var data interface{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&data)
	if err != nil {
		return "", "", fmt.Errorf("getToken: cannot decode data. %s", err.Error())
	}
	cast, ok := data.(map[string]interface{})
	if !ok {
		return "", "", errors.New("getToken: cannot cast data to map[string]interface{}")
	}
	token = fmt.Sprintf("%v", cast["token"])
	sig = fmt.Sprintf("%v", cast["sig"])
	debugPrintf("\nToken: %s. Sig: %s\n", token, sig)
	return token, sig, nil
}

type playlistInfo struct {
	quality string
	link    string
}

func getUsherList(token, sig, vodID string) ([]playlistInfo, error) {
	usherAPI := fmt.Sprintf("http://usher.twitch.tv/vod/%v?nauthsig=%v&nauth=%v&allow_source=true", vodID, sig, token)
	debugPrintf("\nLink to Usher API: %s\n", usherAPI)
	resp, err := http.Get(usherAPI)
	if err != nil {
		return nil, fmt.Errorf("getUsherList: cannot get usher API data. %s", err.Error())
	}
	defer resp.Body.Close()

	resStr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("getUsherList: cannot read response blob. %s", err.Error())
	}
	debugPrintf("\nUsher API response string: %s\n", resStr)
	reg := regexp.MustCompile(regQualityAndM3U8List)
	matches := reg.FindAllString(string(resStr), -1)
	if len(matches) == 0 {
		return nil, errors.New("getUsherList: no matches in M3U8 lists info")
	}
	m := make([]playlistInfo, 0, len(matches))
	for _, str := range matches {
		tmp := strings.Split(str, "\n")
		q := tmp[0][strings.Index(tmp[0], "\"")+1 : strings.LastIndex(tmp[0], "\"")]
		m = append(m, playlistInfo{
			quality: q,
			link:    tmp[1],
		})
	}
	return m, nil
}

func connectTwitch(vodID string) ([]playlistInfo, error) {
	token, sig, err := getToken(vodID)
	if err != nil {
		return nil, err
	}
	pi, err := getUsherList(token, sig, vodID)
	if err != nil {
		return nil, err
	}

	return pi, nil
}

func getTSFromM3U8List(list string) (tsFiles []string, targetDuration int, err error) {
	resp, err := http.Get(list)
	if err != nil {
		return nil, 0, fmt.Errorf("getTSFromM3U8List: cannot retrieve given m3u8 list. %s", err.Error())
	}
	defer resp.Body.Close()

	listStr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("getTSFromM3U8List: cannot read list data. %s", err.Error())
	}
	tdStr := string(listStr)
	bc, ec := strings.Index(tdStr, targetDurationStrBegin)+len(targetDurationStrBegin), strings.Index(tdStr, targetDurationStrEnd)
	targetDuration, err = strconv.Atoi(tdStr[bc:ec])
	if err != nil {
		return nil, 0, fmt.Errorf("getTSFromM3U8List: cannot cast TARGETDURATION to type int. %s", err.Error())
	}
	reg := regexp.MustCompile(regTSfile)
	matches := reg.FindAllString(string(listStr), -1)
	if len(matches) == 0 {
		return nil, 0, errors.New("getTSFromM3U8List: no .ts entries in the list")
	}
	return matches, targetDuration, nil
}

func getM3U8LinkByQiality(pi []playlistInfo, quality string) string {
	list, ok := checkListByQuality(pi, quality)
	if ok {
		tmp := quality
		if quality == defaultQuality {
			tmp = "source"
		}
		fmt.Printf("Downloading in %s quality...\n", tmp)
		return list
	}
	// try to find best quality
	fmt.Printf("No such quality: %s. Trying to find the best one...\n", quality)
	if quality != defaultQuality {
		quality = defaultQuality
		list, ok = checkListByQuality(pi, quality)
	}
	if ok {
		fmt.Println("Found source quality! Downloading in it...")
		return list
	}
	fps, fpsMax, resol, resolMax := 0, 0, 0, 0
	for _, p := range pi {
		tmp := strings.Split(p.quality, "p")
		if len(tmp) > 1 { //fps
			fps, _ = strconv.Atoi(tmp[1])
		} else {
			fps = 0
		}
		resol, _ = strconv.Atoi(tmp[0])
		if resol > resolMax || (fps > fpsMax && resol == resolMax) {
			resolMax, fpsMax, quality = resol, fps, p.quality
		}
	}
	list, ok = checkListByQuality(pi, quality)
	if ok {
		fmt.Printf("Found %s quality as best! Downloading in it...\n", quality)
	} else { // no opts at all
		fmt.Println("No quality options are available for this VOD")
		os.Exit(1)
	}
	return list
}

func checkListByQuality(pi []playlistInfo, quality string) (list string, ok bool) {
	ok = false
	list = ""
	for _, p := range pi {
		if quality == p.quality {
			ok = true
			list = p.link
			break
		}
	}
	return list, ok
}

func downloadTS(path string, base string, vodID string, tsName string, tsNum string, wg *sync.WaitGroup) {
	defer wg.Done()
	<-sem
	tsURL := base + tsName
	retryMax := 5
	var data []byte
LOOP:
	for retry := 0; retry < retryMax; retry++ {
		data = nil
		if retry > 0 {
			debugPrintf("%d try to download %s\n", retry+1, tsName)
		}
		resp, err := http.Get(tsURL)
		if err != nil {
			fatalPrintf(err, "Could not download file %s\n", tsName)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			data, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				fatalPrintf(err, "Could not read file %s. Server returned wrong data\n", tsName)
			}
			debugPrintf("\nDrop %s. Server response with %d code. Read data %s\n", tsName, resp.StatusCode, string(data))
			return
		}
		data, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			if retry == retryMax-1 {
				fatalPrintf(err, "\nCould not download file %s after %d tries\n", tsName, retry)
			} else {
				debugPrintf("\nCould not download %s.\nError: %s\n", tsName, err.Error())
			}
		} else {
			break LOOP
		}
	}
	tsFullOSName := filepath.Join(path, vodID+"_"+tsNum+tsExtension)
	if err := ioutil.WriteFile(tsFullOSName, data, 0400); err != nil {
		fatalPrintf(err, "Could not write file %s in %s\n", tsName, path)
	}
	sem <- struct{}{}
	fmt.Print(".")
}

func calcTSCountByTargetDuration(start, end string, targetDuration int) int {
	ss := convertTimeToSeconds(start)
	es := convertTimeToSeconds(end)
	return ((es - ss) / targetDuration) + 1
}

func calcStartTS(start string, targetDuration int) int {
	ss := convertTimeToSeconds(start)
	return ss / targetDuration
}

func convertTimeToSeconds(timeStr string) int {
	seconds := 0
	if strings.Contains(timeStr, "h") {
		h, err := strconv.Atoi(timeStr[:strings.Index(timeStr, "h")])
		if err != nil {
			fatalPrintf(err, "Cannot convert defined time. Correct format: 1h10m10s or 15m21s or 33s\n")
		}
		seconds += h * 3600
	}
	if strings.Contains(timeStr, "m") {
		if seconds == 0 {
			m, err := strconv.Atoi(timeStr[:strings.Index(timeStr, "m")])
			if err != nil {
				fatalPrintf(err, "Cannot convert defined time. Correct format: 1h10m10s or 15m21s or 33s\n")
			}
			if m > 59 {
				fatalPrintf(errors.New("Overflow time"), "More than 59 minutes in 1 hour! Try once again\n")
			}
			seconds += m * 60
		} else {
			mPos := strings.Index(timeStr, "m")
			hPos := strings.Index(timeStr, "h")
			mins := timeStr[hPos+1 : mPos]
			m, err := strconv.Atoi(mins)
			if err != nil {
				fatalPrintf(err, "Cannot convert defined time. Correct format: 1h10m10s or 15m21s or 33s\n")
			}
			if m > 59 {
				fatalPrintf(errors.New("Overflow time"), "More than 59 minutes in 1 hour! Try once again\n")
			}
			seconds += m * 60
		}
	}
	if strings.Contains(timeStr, "s") {
		if seconds == 0 {
			s, err := strconv.Atoi(timeStr[:strings.Index(timeStr, "s")])
			if err != nil {
				fatalPrintf(err, "Cannot convert defined time. Correct format: 1h10m10s or 15m21s or 33s\n")
			}
			if s > 59 {
				fatalPrintf(errors.New("Overflow time"), "More than 59 seconds in 1 minute! Try once again\n")
			}
			seconds += s
		} else {
			sPos := strings.Index(timeStr, "s")
			mPos := strings.Index(timeStr, "m")
			secs := timeStr[mPos+1 : sPos]
			s, err := strconv.Atoi(secs)
			if err != nil {
				fatalPrintf(err, "Cannot convert defined time. Correct format: 1h10m10s or 15m21s or 33s\n")
			}
			if s > 59 {
				fatalPrintf(errors.New("Overflow time"), "More than 59 seconds in 1 minute! Try once again\n")
			}
			seconds += s
		}
	}
	if seconds == 0 {
		fatalPrintf(errors.New("Unknown time format"), "Cannot convert defined time. Correct format: 1h10m10s or 15m21s or 33s\n")
	}
	return seconds
}

func calcStartTSAndTSCount(start, end string, durations []float64) (tsStart int, tsCountStartEnd int) {
	ss, es := float64(convertTimeToSeconds(start)), float64(convertTimeToSeconds(end))
	partDuration := es - ss
	sum, rest := 0., 0.
	// 1.996; 10.; 10.; 10.; 10.; 10.; 10.
	//    0   1    2    3    4    5    6
	// ss: 13s; es: 40s; part==27s → start == 2. rest == ss(13) - (sum(21.996) - current_dur(10)) == 1.004s to go. sum=0
	// for tsStart < len durs: sum += current_dur; if sum > (rest + part) (28.004) {count == 4 - 2 + 1 (3)}
	for i, d := range durations {
		sum += d
		if sum > ss {
			tsStart, rest = i, (ss - (sum - d))
			break
		}
	}
	sum = 0.
	for i := tsStart; i < len(durations); i++ {
		sum += durations[i]
		if sum > (rest + partDuration) {
			tsCountStartEnd = i - tsStart + 1
		}
	}
	if tsCountStartEnd == 0 {
		tsCountStartEnd = len(durations) - tsStart
	}
	return
}

func getDurationsFromM3U8List(list string) ([]float64, error) {
	resp, err := http.Get(list)
	if err != nil {
		return nil, fmt.Errorf("getDurationsFromM3U8List: cannot retrieve given m3u8 list. %s", err.Error())
	}
	defer resp.Body.Close()

	listStr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("getDurationsFromM3U8List: cannot read list data. %s", err.Error())
	}
	reg := regexp.MustCompile(regDuration)
	matches := reg.FindAllString(string(listStr), -1)
	if len(matches) == 0 {
		return nil, errors.New("getDurationsFromM3U8List: no matches while getting durations")
	}
	durations := make([]float64, 0, len(matches))
	for _, duration := range matches {
		d, err := strconv.ParseFloat(duration, 64)
		if err != nil {
			return nil, fmt.Errorf("getDurationsFromM3U8List: cannot parse duration as float64. %s", err.Error())
		}
		durations = append(durations, d)
	}
	return durations, nil
}

// DownloadVOD download defined VOD from start time to end time with certain quality
// Default value for start "0"; for end "-1"
// Default value for quality if "chunked"
func DownloadVOD(vodID string, start string, end string, quality string) {
	startT := time.Now()
	pi, err := connectTwitch(vodID)
	endT := time.Since(startT)
	if err != nil {
		fatalPrintf(err, "There was an error while connecting to Twitch server\n")
	}
	fmt.Println("Successfully connected to server")
	debugPrintf("\nUsher API playlists info:\n")
	if Debug {
		for _, p := range pi {
			debugPrintf("Quality: %s. m3u8 link: %s\n", p.quality, p.link)
		}
	}
	if TimeF {
		fmt.Printf("Connect time: %f seconds\n", endT.Seconds())
	}

	startT = time.Now()
	fmt.Println("Choosing quality...")
	m3u8link := getM3U8LinkByQiality(pi, quality)
	base := m3u8link[:strings.Index(m3u8link, quality)+len(quality)+1]
	debugPrintf("\nChosen M3U8: %s. Base link: %s\n", m3u8link, base)

	tsList, targetDuration, err := getTSFromM3U8List(m3u8link)
	if err != nil {
		fatalPrintf(err, "There was an error while retreiving data\n")
	}
	debugPrintf("\nList of .ts files: %v\n", tsList)

	tsCountStartEnd, tsStart := 0, 0
	if end != "-1" {
		durations, err := getDurationsFromM3U8List(m3u8link)
		if len(durations) != len(tsList) || err != nil {
			tsStart, tsCountStartEnd = calcStartTS(start, targetDuration), calcTSCountByTargetDuration(start, end, targetDuration)
		} else {
			tsStart, tsCountStartEnd = calcStartTSAndTSCount(start, end, durations)
		}
	} else {
		fmt.Println("Timestamps didn't defined. Downloading full VOD...")
		_, tsCountStartEnd = tsStart, len(tsList)
	}
	debugPrintf("\n.ts files to download: %d. Starting from %d file in m3u8\n", tsCountStartEnd, tsStart)

	pwd := "."
	path, err := ioutil.TempDir(pwd, vodID+"_")
	if err != nil {
		fatalPrintf(err, "Could not create temporary directory\n")
	}
	defer removeTemp(path)
	sCh := make(chan os.Signal, 1)
	signal.Notify(sCh, os.Interrupt, os.Kill)
	go func(path string) {
		<-sCh
		fmt.Println("\nProgram was interrupted by user")
		removeTemp(path)
		os.Exit(1)
	}(path)
	fmt.Printf("Created new temorary directory %s\n", path)
	endT = time.Since(startT)
	if TimeF {
		fmt.Printf("Preparations time: %f seconds\n", endT.Seconds())
	}

	startT = time.Now()
	fmt.Println("Started downloading...")
	var wg sync.WaitGroup
	wg.Add(tsCountStartEnd)
	for i := tsStart; i < (tsCountStartEnd + tsStart); i++ {
		tsName, tsNum := tsList[i], strconv.Itoa(i)
		go downloadTS(path, base, vodID, tsName, tsNum, &wg)
	}
	wg.Wait()
	endT = time.Since(startT)
	if TimeF {
		fmt.Printf("\nDownloading time: %f seconds", endT.Seconds())
	}

	startT = time.Now()
	fmt.Println("\nConverting...")
	err = concatffmpegFiles(path, vodID, tsStart, tsCountStartEnd)
	if err != nil {
		fatalPrintf(err, "FFMPEG could not combine files.\nPlease, remove temporary directory %s by hand\n", path)
	}
	endT = time.Since(startT)
	if TimeF {
		fmt.Printf("Converting time: %f seconds\n", endT.Seconds())
	}
	fmt.Println("Done")
}

func removeTemp(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("removeTemp: could not get abs path for %s. %s", path, err.Error())
	}
	err = os.RemoveAll(abs)
	if err != nil {
		return fmt.Errorf("removeTemp: could not remove directory %s. %s", abs, err.Error())
	}
	fmt.Println("All temporary files and directories were deleted")
	return nil
}

func combineFilesInList(path string, vodID string, tsStart, tsCount int) (string, error) {
	buf := bytes.NewBufferString("")
	for i := tsStart; i < (tsCount + tsStart); i++ {
		fname := fmt.Sprintf("file '%s'\n", filepath.Join(path, vodID+"_"+strconv.Itoa(i)+tsExtension))
		buf.WriteString(fname)
	}
	retList := filepath.Join(path, "_tmp_VOD_list_"+vodID)
	err := ioutil.WriteFile(retList, buf.Bytes(), 0400)
	if err != nil {
		return "", fmt.Errorf("combineFilesInList: could not write in file. %s", err.Error())
	}
	return retList, nil
}

func concatffmpegFiles(path, vodID string, tsStart, tsCount int) error {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	flist, err := combineFilesInList(path, vodID, tsStart, tsCount)
	if err != nil {
		return err
	}
	vodFile := vodID + ".mp4"
	_, err = os.Stat(vodID + ".mp4")
	if err == nil || !os.IsNotExist(err) {
		fname := vodID + "_" + strconv.Itoa(r.Intn(9999)) + ".mp4"
		fmt.Printf("File %s already exists. Created new file %s\n", vodID+".mp4", fname)
		vodFile = fname
	}
	cmdConcat := exec.Command(ffmpegBinary, strings.Fields("-f concat -safe 0 -i "+flist+" -c copy -fflags +genpts -bsf:a aac_adtstoasc "+vodFile)...)
	cmdErr := bytes.NewBuffer(nil)
	cmdConcat.Stderr = cmdErr
	err = cmdConcat.Run()
	if err != nil {
		return fmt.Errorf("concatffmpegFiles: ffmpeg returned error while concat: %s", cmdErr.String())
	}
	return nil
}

func debugPrintf(format string, opts ...interface{}) {
	if Debug {
		if len(format) > 0 {
			fmt.Printf(format, opts...)
		}
	}
}

func fatalPrintf(err error, format string, opts ...interface{}) {
	if len(format) > 0 {
		fmt.Fprintf(os.Stderr, format, opts...)
	}
	debugPrintf(format, err)
	os.Exit(1)
}

// GetVODInfo print only useful data about given VOD ID
// It uses New Twitch API, so be sure that using this function is totally safe for user
func GetVODInfo(vodID string) string {
	var twData struct {
		Data []struct {
			Title       string `json:"title,omitempty"`
			Type        string `json:"type,omitempty"`
			ViewCount   int    `json:"view_count,omitempty"`
			UserID      string `json:"user_id,omitempty"`
			Duration    string `json:"duration,omitempty"`
			CreatedAt   string `json:"created_at,omitempty"`
			Viewable    string `json:"viewable,omitempty"`
			Language    string `json:"language,omitempty"`
			Description string `json:"description,omitempty"`
		} `json:"data"`
	}
	done := make(chan string)
	go printQialityOpts(vodID, done)
	rs := newAPIGetVideo + vodID
	req, _ := http.NewRequest("GET", rs, nil)
	req.Header.Set("Client-ID", twitchClient)
	var c http.Client
	resp, err := c.Do(req)
	if err != nil {
		fatalPrintf(fmt.Errorf("GetVODInfo: cannot retreive VOD info via API. %s", err.Error()), "Could not retrieve data from server\n")
		os.Exit(1)
	}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&twData)
	if err != nil {
		fatalPrintf(fmt.Errorf("GetVODInfo: cannot decode data. %s", err.Error()), "Could not parse recieved data\n")
		os.Exit(1)
	}
	if twData.Data[0].Description == "" {
		twData.Data[0].Description = "Empty"
	}
	t, _ := time.Parse(time.RFC3339, twData.Data[0].CreatedAt)
	tf := fmt.Sprintf("%d/%d/%d %d:%d", t.Month(), t.Day(), t.Year(), t.Hour(), t.Minute())
	ret := fmt.Sprintf("\nTitle: %s\nType: %s\nViews: %d\nStreamer ID: %s\nFull duration: %s\nCreated at: %s\nViewable by: %s\nVideo language: %s\nDescription: %s\n", twData.Data[0].Title, strings.Title(twData.Data[0].Type), twData.Data[0].ViewCount, twData.Data[0].UserID, twData.Data[0].Duration, tf, strings.Title(twData.Data[0].Viewable), strings.Title(twData.Data[0].Language), twData.Data[0].Description)

	retQuality := fmt.Sprintf("\nAvailable quality options:\n%s", <-done)
	return (ret + retQuality)
}

func printQialityOpts(vodID string, done chan<- string) {
	pi, err := connectTwitch(vodID)
	if err != nil {
		fatalPrintf(err, "There was an error while connecting to Twitch server\n")
	}
	buf := bytes.NewBufferString("")
	for _, q := range pi {
		buf.WriteString(q.quality)
		buf.WriteString("\n")
	}
	done <- buf.String()
}
