package wizard

import (
	"testing"

	guided "github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"github.com/zadykian/ws/roles/client/macos/stages/tools"
)

// Picking Cable locks ws client macos apply, and picking another stage locks the one it waits on.
func TestCableLocksWsMacApply(t *testing.T) {
	if got := guided.Locks(&Cable{}); got != stage.Setup {
		t.Errorf("Cable locks %q", got)
	}
	if got := guided.Locks(&tools.Setup{}); got != stage.Homebrew {
		t.Errorf("ws client macos apply locks %q", got)
	}
}
