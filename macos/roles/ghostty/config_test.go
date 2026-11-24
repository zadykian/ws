package ghostty_test

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/ghostty"
)

func TestConfigPathPicksAsGhosttysMenu(t *testing.T) {
	const support = "/Users/v/Library/Application Support/com.mitchellh.ghostty"
	for name, c := range map[string]struct {
		files map[string]string
		want  string
	}{
		"none":                  {nil, support + "/config.ghostty"},
		"the first that holds":  {map[string]string{support + "/config.ghostty": "", "/Users/v/.config/ghostty/config": "a = 1\n"}, "/Users/v/.config/ghostty/config"},
		"the first that exists": {map[string]string{support + "/config": "", "/Users/v/.config/ghostty/config": ""}, support + "/config"},
	} {
		m := effecttest.NewMachine()
		for p, text := range c.files {
			m.Files[p] = []byte(text)
		}
		if got := ghostty.ConfigPath(m, "/Users/v"); got != c.want {
			t.Errorf("%s: config = %q, want %q", name, got, c.want)
		}
	}
}
