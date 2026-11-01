package setup

import "testing"

func same(p string) (string, error) { return p, nil }

func TestGhosttyIncludes(t *testing.T) {
	file := "/h/.config/ghostty/ws.ghostty"
	conf := "/h/.config/ghostty/config"
	for line, want := range map[string]bool{
		"config-file = ws.ghostty":                    true,
		"  config-file=?ws.ghostty \r":                true,
		`config-file = "?ws.ghostty"`:                 true,
		`config-file = ?"ws.ghostty"`:                 true,
		"config-file = ~/.config/ghostty/ws.ghostty":  true,
		"config-file = /h/.config/ghostty/ws.ghostty": true,
		"# config-file = ws.ghostty":                  false,
		"config-file = other.ghostty":                 false,
		"config-file = sub/ws.ghostty":                false,
		"config-files = ws.ghostty":                   false,
		"config-file":                                 false,
	} {
		if got := ghosttyIncludes(conf, "font-size = 1\n"+line+"\n", file, "/h", same); got != want {
			t.Errorf("%q: got %v, want %v", line, got, want)
		}
	}
}

func TestGhosttyIncludesResolvesLinks(t *testing.T) {
	resolve := func(p string) (string, error) {
		if p == "/dots" {
			return "/h/.config/ghostty", nil
		}
		return p, nil
	}
	got := ghosttyIncludes("/h/.config/ghostty/config", "config-file = /dots/ws.ghostty",
		"/h/.config/ghostty/ws.ghostty", "/h", resolve)
	if !got {
		t.Error("a path through a link did not count")
	}
}
