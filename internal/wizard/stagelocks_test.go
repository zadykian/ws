package wizard

import (
	"testing"

	guided "github.com/zadykian/ws/engine/core/wizard"
)

// Picking Cable locks ws client macos apply, and picking another stage locks the one it waits on.
func TestCableLocksWsMacApply(t *testing.T) {
	if got := guided.Locks(&Cable{}); got != setupStage {
		t.Errorf("Cable locks %q", got)
	}
	if got := guided.Locks(&Setup{}); got != homebrewStage {
		t.Errorf("ws client macos apply locks %q", got)
	}
}
