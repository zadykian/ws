package cli

import "testing"

func TestReleasesURL(t *testing.T) {
	for set, want := range map[string]string{"": "https://github.com/zadykian/ws/releases", "http://127.0.0.1:22080": "http://127.0.0.1:22080"} {
		if got := releasesURL(func(string) string { return set }); got != want {
			t.Errorf("WS_RELEASES_URL=%q: releasesURL = %q, want %q", set, got, want)
		}
	}
}
