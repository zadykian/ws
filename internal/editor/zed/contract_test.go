package zed

import (
	"testing"

	"github.com/zadykian/ws/internal/editor"
)

// TestEveryActionHasATargetOrAReason is the adapter's contract: an action with neither fails.
func TestEveryActionHasATargetOrAReason(t *testing.T) {
	keymap, err := editor.LoadKeymap()
	if err != nil {
		t.Fatal(err)
	}
	targets, err := editor.LoadTargets(actionFiles, "actions/*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := keymap.Check(targets); err != nil {
		t.Fatal(err)
	}
}
