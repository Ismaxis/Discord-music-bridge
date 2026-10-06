package main

import (
	"reflect"
	"testing"
)

func TestIsYouTubeURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://www.youtube.com/watch?v=video", true},
		{"https://youtu.be/video", true},
		{"http://music.youtube.com/watch?v=video", true},
		{"https://youtube.com.evil.example/watch?v=video", false},
		{"https://youtube.com@evil.example/watch?v=video", false},
		{"file:///youtube.com/video", false},
		{"not a URL", false},
		{"https://youtube.com:bad/watch?v=video", false},
	}
	for _, test := range tests {
		t.Run(test.url, func(t *testing.T) {
			if got := isYouTubeURL(test.url); got != test.want {
				t.Errorf("isYouTubeURL(%q) = %t, want %t", test.url, got, test.want)
			}
		})
	}
}

func TestYouTubeDLArgs(t *testing.T) {
	tests := []struct {
		name string
		cfg  config
		want []string
	}{
		{
			name: "defaults",
			cfg:  config{},
			want: []string{
				"--no-playlist", "--no-progress", "--max-filesize", "100M",
				"--format", "bestaudio", "--output", "source.%(ext)s",
				"https://youtube.com/watch?v=test",
			},
		},
		{
			name: "browser cookies and runtime",
			cfg:  config{youtubeCookiesBrowser: "chrome", youtubeJSRuntime: "deno"},
			want: []string{
				"--no-playlist", "--no-progress", "--max-filesize", "100M",
				"--format", "bestaudio", "--output", "source.%(ext)s",
				"--cookies-from-browser", "chrome", "--js-runtimes", "deno",
				"https://youtube.com/watch?v=test",
			},
		},
		{
			name: "cookies file",
			cfg:  config{youtubeCookiesFile: `C:\secure\cookies.txt`},
			want: []string{
				"--no-playlist", "--no-progress", "--max-filesize", "100M",
				"--format", "bestaudio", "--output", "source.%(ext)s",
				"--cookies", `C:\secure\cookies.txt`,
				"https://youtube.com/watch?v=test",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := youtubeDLArgs("https://youtube.com/watch?v=test", "source.%(ext)s", test.cfg)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("youtubeDLArgs() = %#v, want %#v", got, test.want)
			}
		})
	}
}
