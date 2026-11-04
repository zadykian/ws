package fresh

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/editor"
)

func TestNoKeyConflictsWithItsTerminalEquivalent(t *testing.T) {
	bound := map[string]string{}
	for _, b := range typed(t).Maps["f12"].Bindings {
		if len(b.Keys) == 0 {
			bound[b.When+" "+strings.Join(b.Modifiers, "+")+"+"+b.Key] = b.Action
		}
	}
	for _, when := range []string{"normal", "file_explorer", "terminal"} {
		for _, pair := range [][2]string{
			{"ctrl+/", "ctrl+7"}, {"ctrl+Backspace", "ctrl+h"}, {"ctrl+Space", "ctrl+@"}, {"ctrl+-", "ctrl+_"},
		} {
			one, two := bound[when+" "+pair[0]], bound[when+" "+pair[1]]
			if one != two && one != "" && two != "" {
				t.Errorf("%s: %s is %s, %s is %s", when, pair[0], one, pair[1], two)
			}
		}
	}
	if bound["normal ctrl+h"] != "" || bound["normal ctrl+Backspace"] != "" {
		t.Error("a map's Ctrl+H or Ctrl+Backspace warns at each start, against Fresh's default")
	}
}

func TestCtrlHBindsOutsideTheMap(t *testing.T) {
	outside := typed(t).Keybindings
	if len(outside) != 1 || outside[0].Action != "replace" || outside[0].When != "normal" {
		t.Fatalf("%+v", outside)
	}
}

// TestEachArrivalIsBound renders the keymap: a chord that terminals rewrite gets a binding under
// the key it arrives as, for its action. Without one, the chord lands on no action through tmux.
func TestEachArrivalIsBound(t *testing.T) {
	keymap, err := editor.LoadKeymap()
	if err != nil {
		t.Fatal(err)
	}
	targets, err := editor.LoadTargets(actionFiles, "actions/*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	all, err := bindings(keymap, targets)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, scope := range editor.Scopes {
		for _, b := range keymap[scope] {
			target := targets[b.Action]
			for _, chord := range b.Keys {
				text, arrives := arrivals[chord.String()]
				if !arrives || target.To == "" {
					continue
				}
				found++
				other, err := editor.ParseChord(text)
				if err != nil {
					t.Fatal(err)
				}
				for _, sequence := range expand(other) {
					want := binding{Action: target.To, When: contexts[scope]}
					if len(sequence) == 1 {
						want.Key, want.Modifiers = sequence[0].Key, sequence[0].Modifiers
					} else {
						want.Keys = sequence
					}
					if !slices.ContainsFunc(all, func(got binding) bool {
						return got.Action == want.Action && got.When == want.When && got.Key == want.Key &&
							slices.Equal(got.Modifiers, want.Modifiers) && slices.EqualFunc(got.Keys, want.Keys,
							func(a, b press) bool { return a.Key == b.Key && slices.Equal(a.Modifiers, b.Modifiers) })
					}) {
						t.Errorf("%s arrives as %s, which binds nothing for %s", chord, text, target.To)
					}
				}
			}
		}
	}
	if found != len(arrivals) {
		t.Errorf("%d of %d arrivals are chords of F12", found, len(arrivals))
	}
}

func TestTheRenderHoldsItsBindings(t *testing.T) {
	c := typed(t)
	when := map[string]int{}
	for _, b := range c.Maps["f12"].Bindings {
		when[b.When]++
	}
	want := map[string]int{"normal": 158, "file_explorer": 13, "terminal": 27}
	if len(when) != len(want) || when["normal"] != 158 || when["file_explorer"] != 13 || when["terminal"] != 27 ||
		len(c.Keybindings) != 1 {
		t.Errorf("by when %v, %d outside the map; want %v and 1", when, len(c.Keybindings), want)
	}
}
