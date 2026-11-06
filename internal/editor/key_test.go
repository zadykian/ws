package editor

import "testing"

func TestAChordRoundTrips(t *testing.T) {
	for _, text := range []string{"ctrl+k ctrl+c", "ctrl+alt+shift+,", "cmd+a", "alt+shift+[", "f12",
		"ctrl+numpad* 1", "shift+escape", "ctrl+alt+pagedown"} {
		chord, err := ParseChord(text)
		if err != nil {
			t.Errorf("%s: %v", text, err)
			continue
		}
		if got := chord.String(); got != text {
			t.Errorf("%s reads back as %s", text, got)
		}
	}
}

func TestAChordOutOfTheSpellingFails(t *testing.T) {
	for _, text := range []string{"", "ctrl+", "ctrl+?", "alt+{", "shift+ctrl+a", "ctrl-a", "ctrl+A", "numpad*x"} {
		if chord, err := ParseChord(text); err == nil {
			t.Errorf("%q parses as %v", text, chord)
		}
	}
}
