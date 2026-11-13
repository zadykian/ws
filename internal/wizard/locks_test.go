package wizard

import "testing"

// Picking Cable locks ws mac apply, and picking another stage locks the one it waits on.
func TestCableLocksWsMacApply(t *testing.T) {
	if got := Locks(&Cable{}); got != setupStage {
		t.Errorf("Cable locks %q", got)
	}
	if got := Locks(&Setup{}); got != homebrewStage {
		t.Errorf("ws client macos apply locks %q", got)
	}
}
