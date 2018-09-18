package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/zerospiel/ttvldr/downloader"
)

const (
	regCheckCorrectArg = "(\\s|https:\\/\\/www\\.|^|www\\.)twitch\\.tv\\/videos\\/(\\d+){9}$"
)

var debug, timeF bool

// TODO
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
	quality := flag.String("quality", defaultQuality, "Defines quality of VOD. 'Chunked' is the source quality")
	flag.BoolVar(&debug, "debug", false, "If set — output debug info")
	flag.BoolVar(&timeF, "time", false, "If set — shows elapsed time for each period of work")
	info := flag.Bool("info", false, "Shows full info about VOD and quality options")
	cpuprofile := flag.String("cpuprofile", "", "Dump CPU usage profile to a certain file to further <go tool pprof>")
	memprofile := flag.String("memprofile", "", "Dump RAM usage profile to a certain file to further <go tool pprof>")
	flag.Parse()
	downloader.Debug = debug
	downloader.TimeF = timeF

	args := flag.Args()
	if len(args) != 1 {
		usage()
		os.Exit(1)
	}

	vodID := getVODFromStdin(args[0])
	if vodID == defaultVOD {
		usage()
		os.Exit(1)
	}

	if *info {
		fmt.Print(downloader.GetVODInfo(vodID))
		os.Exit(0)
	}

	startT := time.Now()
	if defaultSE == *start || defaultSE == *end {
		downloader.DownloadVOD(vodID, "0", "-1", *quality)
		if *cpuprofile != "" {
			f, err := os.Create(*cpuprofile)
			if err != nil {
				panic(err)
			}
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}
		if *memprofile != "" {
			f, err := os.Create(*memprofile)
			if err != nil {
				panic(err)
			}
			pprof.WriteHeapProfile(f)
			f.Close()
		}
	} else {
		downloader.DownloadVOD(vodID, *start, *end, *quality)
		if *cpuprofile != "" {
			f, err := os.Create(*cpuprofile)
			if err != nil {
				panic(err)
			}
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}
		if *memprofile != "" {
			f, err := os.Create(*memprofile)
			if err != nil {
				panic(err)
			}
			pprof.WriteHeapProfile(f)
			f.Close()
		}
	}
	endT := time.Since(startT)
	if timeF {
		fmt.Printf("Total elapsed time: %f minutes\n", endT.Minutes())
	}
}

func usage() {
	fmt.Println("Wrong input. Usage: ttvldr <flags> https://www.twitch.tv/videos/123456789. Check -help option for more information")
}

func getVODFromStdin(input string) string {
	reg := regexp.MustCompile(regCheckCorrectArg)
	if reg.MatchString(input) {
		return input[strings.Index(input, "videos")+len("videos")+1:]
	}
	return "-1"
}
