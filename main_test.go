package main

import (
	"testing"
)

func TestGetVODFromStdin(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{input: "twitch.tv/video/12345678", want: "-1"},
		{input: "www.twitch.tv/video/12345678", want: "-1"},
		{input: "https://twitch.tv/video/12345678", want: "-1"},
		{input: "twitch.tv/videos/12345678", want: "-1"},
		{input: "www.twitch.tv/videos/12345678", want: "-1"},
		{input: "https://www.twitch.tv/videos/12345678", want: "-1"},
		{input: "twitch.tv/videos/123456789", want: "123456789"},
		{input: "www.twitch.tv/videos/123456789", want: "123456789"},
		{input: "https://www.twitch.tv/videos/123456789", want: "123456789"},
		{input: "http://www.twitch.tv/videos/123456789", want: "123456789"},
		{input: "http://foobar.com", want: "-1"},
		{input: "foobar.com", want: "-1"},
		{input: "www.foobar.com", want: "-1"},
		{input: "some random string that may contain twitch or twitch.tv or even videos or some ID 123456789", want: "-1"},
		{input: "https://www.twitch.tv/videos/309711819", want: "309711819"},
		{input: "", want: "-1"},
	}
	for _, c := range cases {
		got := getVODFromStdin(c.input)
		if got != c.want {
			t.Errorf("getVODFromStdin: failed test. got: %s; want: %s", got, c.want)
		}
	}
}
