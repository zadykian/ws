package ghostty

import (
	"errors"
	"testing"
)

func same(p string) (string, error) { return p, nil }

func TestGhosttyIncludes(t *testing.T) {
	file := "/etc/ws/ghostty"
	conf := "/h/.config/ghostty/config"
	for line, want := range map[string]bool{
		"config-file = /etc/ws/ghostty":     true,
		"  config-file=?/etc/ws/ghostty \r": true,
		`config-file = "?/etc/ws/ghostty"`:  true,
		`config-file = ?"/etc/ws/ghostty"`:  true,
		"# config-file = /etc/ws/ghostty":   false,
		"config-file = /etc/ws/other":       false,
		"config-file = /opt/ws/ghostty":     false,
		"config-file = ghostty":             false,
		"config-files = /etc/ws/ghostty":    false,
		"config-file":                       false,
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
	got := ghosttyIncludes("/h/.config/ghostty/config", "config-file = /dots/ghostty",
		"/h/.config/ghostty/ghostty", "/h", resolve)
	if !got {
		t.Error("a path through a link did not count")
	}
}

func TestGhosttyIncludesComparesCleanedPathsWhereTheDirectoryIsMissing(t *testing.T) {
	missing := func(p string) (string, error) {
		if p == "/etc/ws" {
			return "", errors.New("no such directory")
		}
		return p, nil
	}
	conf := "/h/.config/ghostty/config"
	for line, want := range map[string]bool{
		"config-file = ?/etc/ws/ghostty":       true,
		"config-file = /etc//ws/../ws/ghostty": true,
		"config-file = /opt/ws/ghostty":        false,
		"config-file = /etc/ws/other":          false,
	} {
		if got := ghosttyIncludes(conf, line+"\n", "/etc/ws/ghostty", "/h", missing); got != want {
			t.Errorf("%q: got %v, want %v", line, got, want)
		}
	}
}
