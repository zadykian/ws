package fresh

import (
	"regexp"
	"slices"
	"testing"
)

// Fresh reads a misspelt when as normal, drops a key it can't name and ignores a modifier it
// doesn't know, so that the plain key binds. Each spelling is checked here.
var (
	whens     = []string{"normal", "file_explorer", "terminal"}
	modifiers = []string{"ctrl", "alt", "shift", "super"}
	keyNames  = regexp.MustCompile(`^(F([1-9]|1[0-9]|2[0-4])|Up|Down|Left|Right|Home|End|PageUp|PageDown|` +
		`Enter|Esc|Tab|Space|Backspace|Delete|[^A-Z\s])$`)
)

func TestEachBindingIsSpeltAsFreshReadsIt(t *testing.T) {
	_, c := load(t)
	all := slices.Clone(c.Keybindings)
	for _, m := range c.Maps {
		all = append(all, m.Bindings...)
	}
	for _, b := range all {
		presses := b.Keys
		if len(presses) == 0 {
			presses = []press{b.press}
		}
		if !slices.Contains(whens, b.When) {
			t.Errorf("%s: when %q", b.spellAll(), b.When)
		}
		for _, p := range presses {
			if !keyNames.MatchString(p.Key) {
				t.Errorf("%s: key %q isn't one Fresh names", b.spellAll(), p.Key)
			}
			for _, m := range p.Modifiers {
				if !slices.Contains(modifiers, m) {
					t.Errorf("%s: modifier %q isn't one Fresh knows", b.spellAll(), m)
				}
			}
		}
	}
}
