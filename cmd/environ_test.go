package cmd

import "testing"

func TestConfPath(t *testing.T) {
	for conf, want := range map[string]string{"": "/h/.config/ws/mac.conf", "/x/m.conf": "/x/m.conf", "m.conf": "/w/m.conf"} {
		if got := confPath(func(k string) string { return map[string]string{"HOME": "/h", "WS_CONF": conf}[k] }, "/w"); got != want {
			t.Errorf("WS_CONF=%q: confPath = %q, want %q", conf, got, want)
		}
	}
}

func TestReleasesURL(t *testing.T) {
	for set, want := range map[string]string{"": "https://github.com/zadykian/ws/releases", "http://127.0.0.1:22080": "http://127.0.0.1:22080"} {
		if got := releasesURL(func(string) string { return set }); got != want {
			t.Errorf("WS_RELEASES_URL=%q: releasesURL = %q, want %q", set, got, want)
		}
	}
}
