package fresh

import (
	"strings"
	"testing"
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
