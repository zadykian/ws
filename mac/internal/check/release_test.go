package check

import "testing"

func TestRelease(t *testing.T) {
	for s, want := range map[string]bool{
		"v0.0.0": true, "v1.20.3": true, "v10.0.1": true,
		"": false, "1.2.3": false, "v1.2": false, "v01.2.3": false, "v1.2.3\n": false,
		"v0.0.1-0.20261030120000-abcdef123456": false, "v1.2.3+dirty": false, "devel": false,
	} {
		if got := Release(s); got != want {
			t.Errorf("Release(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestReleasesURL(t *testing.T) {
	for s, want := range map[string]bool{
		"https://github.com/zadykian/ws/releases": true,
		"http://127.0.0.1:22080":                  true,
		"http://127.0.0.1:22080/dir":              true,
		"http://github.com/zadykian/ws/releases":  false,
		"http://localhost:22080":                  false,
		"https://example.org/a b":                 false,
		"https://example.org/'x'":                 false,
		"https://example.org/$(id)":               false,
		"https://":                                false,
		"":                                        false,
	} {
		if got := ReleasesURL(s); got != want {
			t.Errorf("ReleasesURL(%q) = %v, want %v", s, got, want)
		}
	}
}
