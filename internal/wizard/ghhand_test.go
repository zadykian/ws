package wizard

import (
	"context"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestGhHand(t *testing.T) {
	for cred, want := range map[string]int{"!/bin/gh auth git-credential": 0, "osxkeychain": 1} {
		m := effecttest.NewMachine()
		m.Results["git config --global --includes --get-all credential.https://github.com.helper"] =
			effecttest.Result{Out: []byte(cred + "\n")}
		hand := &Hand{}
		(&Gh{GhCLI: GhCLI{M: m}}).Hand(context.Background(), hand)
		if len(hand.lines) != want {
			t.Errorf("%s: %d lines in the list; want %d", cred, len(hand.lines), want)
		}
	}
}
