package downloader

import (
	"fmt"
	"strings"
	"testing"
)

var (
	vodID = "309711819" //this is a HL so it's 99.99% never be deleted
)

func TestGetM3U8LinkByQiality(t *testing.T) {
	pi := []playlistInfo{
		{quality: "chunked", link: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/chunked/highlight-309711819.m3u8"},
		{quality: "720p60", link: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/720p60/highlight-309711819.m3u8"},
		{quality: "720p30", link: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/720p30/highlight-309711819.m3u8"},
		{quality: "480p30", link: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/480p30/highlight-309711819.m3u8"},
		{quality: "360p30", link: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/360p30/highlight-309711819.m3u8"},
		{quality: "160p30", link: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/160p30/highlight-309711819.m3u8"},
	}
	cases := []struct {
		input, want string
	}{
		{input: "chunked", want: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/chunked/highlight-309711819.m3u8"},
		{input: "720p60", want: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/720p60/highlight-309711819.m3u8"},
		{input: "720p30", want: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/720p30/highlight-309711819.m3u8"},
		{input: "480p30", want: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/480p30/highlight-309711819.m3u8"},
		{input: "360p30", want: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/360p30/highlight-309711819.m3u8"},
		{input: "160p30", want: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/160p30/highlight-309711819.m3u8"},
	}
	for _, c := range cases {
		got := getM3U8LinkByQiality(pi, c.input)
		if got != c.want {
			t.Errorf("getM3U8LinkByQiality: test failed. got: %s. want: %s", got, c.want)
		}
	}
}

func TestGetToken(t *testing.T) {
	sigWant := "cee2dbb02315a633a1d35f1ee62c83742fd28fe6"
	_, sig, err := getToken(vodID)
	if err != nil || len(sig) != len(sigWant) {
		t.Errorf("getToken: test failed. want: %s. got: %s. err: %v", sigWant, sig, err)
	}
}

func TestGetUsherList(t *testing.T) {
	token, sig, _ := getToken(vodID)
	want := []playlistInfo{
		{quality: "chunked", link: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/chunked/highlight-309711819.m3u8"},
		{quality: "720p60", link: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/720p60/highlight-309711819.m3u8"},
		{quality: "720p30", link: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/720p30/highlight-309711819.m3u8"},
		{quality: "480p30", link: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/480p30/highlight-309711819.m3u8"},
		{quality: "360p30", link: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/360p30/highlight-309711819.m3u8"},
		{quality: "160p30", link: "http://fastly.vod.hls.ttvnw.net/2268723385e60269b21f_baggins_tv_30344829920_965177301/160p30/highlight-309711819.m3u8"},
	}
	got, err := getUsherList(token, sig, vodID)
	if err != nil {
		t.Errorf("getUsherList: test failed. got an error: %s", err.Error())
	}
	for i, g := range got {
		if g.quality != want[i].quality {
			t.Errorf("getUsherList: test failed. got: %v. want: %v", g.quality, want[i].quality)
		}
		// server may change, so have to check if suffixes are the same
		if g.link[strings.Index(g.link, "2268723385e60269b21f"):] != want[i].link[strings.Index(want[i].link, "2268723385e60269b21f"):] {
			t.Errorf("getUsherList: test failed. got: %v. want: %v", g.link, want[i].link)
		}
	}
}

func ExampleGetVODInfo() {
	fmt.Println(GetVODInfo(vodID))

	// Output:
	// Title: Keep On Rolling Rolling Rolling
	// Type: Highlight
	// Views: 19
	// Streamer ID: 116245074
	// Full duration: 17m14s
	// Created at: 9/13/2018 21:47
	// Viewable by: Public
	// Video language: En
	// Description: Empty

	// Available quality options:
	// chunked
	// 720p60
	// 720p30
	// 480p30
	// 360p30
	// 160p30

}
