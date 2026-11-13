package tools

import "testing"

func TestLatestTag(t *testing.T) {
	for in, want := range map[string]string{
		"http://127.0.0.1:22080/tag/v1.2.3":                    "v1.2.3",
		"https://github.com/zadykian/ws/releases/tag/v0.10.0":  "v0.10.0",
		"https://github.com/zadykian/ws/releases/tag/v1.2.3\n": "v1.2.3",
		"https://github.com/zadykian/ws/releases/tag/v01.2.3":  "",
		"https://github.com/zadykian/ws/releases/tag/devel":    "",
		"https://github.com/zadykian/ws/releases":              "",
		"": "",
	} {
		if got, ok := latestTag(in); got != want || ok != (want != "") {
			t.Errorf("latestTag(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
}
