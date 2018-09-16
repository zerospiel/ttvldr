package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/zerospiel/ttvldr/downloader"
)

const (
	regCheckCorrectArg = "(\\s|https:\\/\\/www\\.|^|www\\.)twitch\\.tv\\/videos\\/(\\d+)$"
)

var debug, timeF bool

// TODO
// Optimize RAM usage
// Write tests for API connections, downloading TS, downloading VOD
// Write readme
// Write errors to Stderr
// DO todos
func main() {
	defaultVOD := "-1"
	defaultSE := "-1"
	defaultQuality := "chunked"
	start := flag.String("start", defaultSE, "Start VOD with a certain time, e.g. 0h20m19s")
	end := flag.String("end", defaultSE, "End VOD with a certain time, e.g. 3h04m0s")
	quality := flag.String("quality", defaultQuality, "Defines quality of VOD. Default is best available quality")
	flag.BoolVar(&debug, "debug", false, "If set — output debug info")
	flag.BoolVar(&timeF, "time", false, "If set — shows elapsed time for each period of work")
	info := flag.Bool("info", false, "Shows full info about VOD and quality options")
	flag.Parse()
	downloader.Debug = timeF
	downloader.TimeF = timeF

	// TODO check ffmpeg in a directory

	args := flag.Args()
	if len(args) != 1 {
		usage()
		os.Exit(1)
	}

	vodID := getVODFromStdin(args[0])
	// TODO check VOD ID length maybe?
	if vodID == defaultVOD {
		usage()
		os.Exit(1)
	}

	// TODO end up with info func
	if *info {
		fmt.Println("info currently unavailable")
		// d, _ := getVODinfo(vodID)
		// fmt.Println(d)
		os.Exit(0)
	}

	startT := time.Now()
	if defaultSE == *start || defaultSE == *end {
		downloader.DownloadVOD(vodID, "0", "-1", *quality)
	} else {
		downloader.DownloadVOD(vodID, *start, *end, *quality)
	}
	endT := time.Since(startT)
	if timeF {
		fmt.Printf("Total elapsed time: %f minutes\n", endT.Minutes())
	}
}

func getVODFromStdin(input string) string {
	reg := regexp.MustCompile(regCheckCorrectArg)
	if reg.MatchString(input) {
		return input[strings.Index(input, "videos")+len("videos")+1:]
	}
	return "-1"
}

func usage() {
	fmt.Println("Wrong input. Usage: ttvldr <flags> https://www.twitch.tv/videos/123456789. Check -help option for more information")
}
