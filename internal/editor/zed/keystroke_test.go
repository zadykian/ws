package zed

import (
	"testing"

	"github.com/zadykian/ws/internal/editor"
)

func TestAChordIsSpeltAsZedSpellsIt(t *testing.T) {
	for text, want := range map[string]string{
		"ctrl+alt+shift+,": "ctrl-alt-<",
		"ctrl+shift+/":     "ctrl-?",
		"alt+shift+[":      "alt-{",
		"alt+shift+0":      "alt-)",
		"ctrl+shift+a":     "ctrl-shift-a",
		"ctrl+k ctrl+c":    "ctrl-k ctrl-c",
		"ctrl+numpad* 1":   "ctrl-* 1",
		"shift+f6":         "shift-f6",
	} {
		chord, err := editor.ParseChord(text)
		if err != nil {
			t.Fatal(err)
		}
		if got := keystroke(chord); got != want {
			t.Errorf("%s = %s; want %s", text, got, want)
		}
	}
}
