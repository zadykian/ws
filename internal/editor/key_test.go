package editor

import "testing"

func TestAChordOutOfTheSpellingFails(t *testing.T) {
	for _, text := range []string{"", "ctrl+", "ctrl+?", "alt+{", "shift+ctrl+a", "ctrl-a", "ctrl+A", "numpad*x"} {
		if chord, err := ParseChord(text); err == nil {
			t.Errorf("%q parses as %v", text, chord)
		}
	}
}
