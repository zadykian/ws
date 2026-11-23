package cli

import "testing"

func TestReleasesURL(t *testing.T) {
	for name, tc := range map[string]struct{ set, want string }{
		"GitHub's by default": {"", "https://github.com/zadykian/ws/releases"},
		"the setting's":       {"http://127.0.0.1:22080", "http://127.0.0.1:22080"},
	} {
		t.Run(name, func(t *testing.T) {
			got := releasesURL(func(k string) string {
				if k == "WS_RELEASES_URL" {
					return tc.set
				}
				return ""
			})
			if got != tc.want {
				t.Errorf("releasesURL = %q, want %q", got, tc.want)
			}
		})
	}
}
