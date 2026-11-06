package editor

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Stroke is a physical key and the modifiers held with it. A shifted sign is its key with Shift, as
// a terminal and a Mac spell it differently and each adapter spells it its own way.
type Stroke struct {
	Ctrl, Alt, Cmd, Shift bool
	Key                   string
}

// Chord is a binding's strokes, one after the other.
type Chord []Stroke

// namedKeys lists the keys with a name. numpad* is the numpad's, which F12 binds apart from Shift+8.
var namedKeys = []string{
	"up", "down", "left", "right", "home", "end", "pageup", "pagedown", "enter", "escape", "tab",
	"space", "backspace", "delete", "numpad*",
}

// knownKey says whether a stroke may name the key: a letter, a digit, an unshifted sign, F1 to F24
// or a named key. A shifted sign is refused, as an adapter spells it from its unshifted key.
func knownKey(key string) bool {
	if len(key) == 1 {
		return strings.ContainsAny(key, "abcdefghijklmnopqrstuvwxyz0123456789-`=[]\\;',./")
	}
	if number, found := strings.CutPrefix(key, "f"); found {
		n, err := strconv.Atoi(number)
		return err == nil && n >= 1 && n <= 24
	}
	return slices.Contains(namedKeys, key)
}

// ParseChord reads a chord: strokes split by a space, a stroke as ctrl+alt+cmd+shift+KEY, its
// modifiers in that order.
func ParseChord(text string) (Chord, error) {
	var chord Chord
	for part := range strings.SplitSeq(text, " ") {
		stroke, err := parseStroke(part)
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", text, err)
		}
		chord = append(chord, stroke)
	}
	return chord, nil
}

func parseStroke(text string) (Stroke, error) {
	var s Stroke
	rest := text
	for _, m := range []struct {
		name string
		flag *bool
	}{{"ctrl+", &s.Ctrl}, {"alt+", &s.Alt}, {"cmd+", &s.Cmd}, {"shift+", &s.Shift}} {
		if after, found := strings.CutPrefix(rest, m.name); found {
			*m.flag, rest = true, after
		}
	}
	if !knownKey(rest) {
		return Stroke{}, fmt.Errorf("no key %q", rest)
	}
	s.Key = rest
	return s, nil
}

// String spells the stroke as the data does.
func (s Stroke) String() string {
	var b strings.Builder
	for _, m := range []struct {
		name string
		on   bool
	}{{"ctrl+", s.Ctrl}, {"alt+", s.Alt}, {"cmd+", s.Cmd}, {"shift+", s.Shift}} {
		if m.on {
			b.WriteString(m.name)
		}
	}
	b.WriteString(s.Key)
	return b.String()
}

// String spells the chord as the data does.
func (c Chord) String() string {
	parts := make([]string, len(c))
	for i, s := range c {
		parts[i] = s.String()
	}
	return strings.Join(parts, " ")
}

// UnmarshalYAML reads a chord from its text.
func (c *Chord) UnmarshalYAML(node *yaml.Node) error {
	var text string
	if err := node.Decode(&text); err != nil {
		return err
	}
	chord, err := ParseChord(text)
	if err != nil {
		return err
	}
	*c = chord
	return nil
}
