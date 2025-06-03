package bot_test

import (
	"testing"

	"gh.tarampamp.am/video-dl-bot/internal/bot"
)

func TestExtractLink(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		giveText string
		wantUrl  string // empty = expect an error
	}{
		"markdown http link": {
			giveText: "Download [here](http://example.com/file.zip)",
			wantUrl:  "http://example.com/file.zip",
		},
		"markdown https link": {
			giveText: "Click [this link](https://secure.com/path/to/file)",
			wantUrl:  "https://secure.com/path/to/file",
		},
		"raw https url": {
			giveText: "Here is the link: https://example.org/data.json",
			wantUrl:  "https://example.org/data.json",
		},
		"raw http url": {
			giveText: "Check http://foo.bar/test.txt",
			wantUrl:  "http://foo.bar/test.txt",
		},
		"bare domain with path": {
			giveText: "Try example.net/asset.png for the image.",
			wantUrl:  "https://example.net/asset.png",
		},
		"bare domain without path": {
			giveText: "Visit example.io/foo for more info",
			wantUrl:  "https://example.io/foo",
		},
		"multiple links, prefer markdown": {
			giveText: "See [doc](https://docs.org/manual) or https://alt.org/doc",
			wantUrl:  "https://docs.org/manual",
		},
		"multiple raw links, first wins": {
			giveText: "Try https://first.com/foo and then http://second.com/bar",
			wantUrl:  "https://first.com/foo",
		},
		"IP address link": {
			giveText: "Direct IP: http://192.168.0.1/file.tar.gz 😁",
			wantUrl:  "http://192.168.0.1/file.tar.gz",
		},
		"localhost link": {
			giveText: "Localhost\n test: \t\n\n\thttp://localhost:8080/page\t\n",
			wantUrl:  "http://localhost:8080/page",
		},
		"invalid domain suffix": {
			giveText: "Go to example.invalid/path",
			wantUrl:  "https://example.invalid/path", // still matches by regex, valid from URL standpoint
		},
		"domain with dash": {
			giveText: "dash-site.com/files.zip",
			wantUrl:  "https://dash-site.com/files.zip",
		},
		"full youtube link": {
			giveText: "https://www.youtube.com/watch?v=2PuFyjAs7JA&pp=ygUKdGVzdCB2aWRlb9IHCQmwCQGHKiGM7w%3D%3D",
			wantUrl:  "https://www.youtube.com/watch?v=2PuFyjAs7JA&pp=ygUKdGVzdCB2aWRlb9IHCQmwCQGHKiGM7w%3D%3D",
		},
		"shorten youtube link with an anchor": {
			giveText: "https://youtu.be/2PuFyjAs7JA?si=MlSiEJ6yBetaT2Z8",
			wantUrl:  "https://youtu.be/2PuFyjAs7JA?si=MlSiEJ6yBetaT2Z8",
		},
		"shorten youtube link without protocol": {
			giveText: "goto youtu.be/2PuFyjAs7JA to watch the video",
			wantUrl:  "https://youtu.be/2PuFyjAs7JA", // defaults to https
		},

		"empty text": {
			giveText: "",
			wantUrl:  "",
		},
		"malformed markdown link": {
			giveText: "Check [this link](not-a-url)",
			wantUrl:  "",
		},
		"text with no links": {
			giveText: "There is no link here, just text.",
			wantUrl:  "",
		},
		"text with dots": {
			giveText: "aaa.bbb",
			wantUrl:  "",
		},
		"test http domain": {
			giveText: "https://666",
			wantUrl:  "",
		},
		"cyr domain": {
			giveText: "открой запретограм.рф/видео чтобы посмотреть",
			wantUrl:  "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			res, err := bot.ExtractLink(tc.giveText)

			if tc.wantUrl == "" {
				if err == nil {
					t.Errorf("expected an error, got nil")
				}

				if res != nil {
					t.Errorf("expected no URL, got %q", res)
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if res == nil {
				t.Errorf("expected a URL, got nil")

				return
			}

			if want, got := tc.wantUrl, res.String(); want != got {
				t.Errorf("expected URL %q, got %q", want, got)
			}
		})
	}
}
