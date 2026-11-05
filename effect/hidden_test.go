package effect_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect"
)

func TestAskHiddenNeedsATerminal(t *testing.T) {
	var out bytes.Buffer
	term := effect.NewTerminal(strings.NewReader("secret\n"), &out)
	_, err := term.AskHidden(context.Background(), "Password")
	if err == nil || !strings.Contains(err.Error(), "no terminal") {
		t.Fatalf("AskHidden = %v; want no terminal", err)
	}
	if out.Len() != 0 {
		t.Errorf("AskHidden printed %q", out.String())
	}
}

func TestAskHiddenStopsOnAnEndedContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	term := effect.NewTerminal(strings.NewReader(""), &bytes.Buffer{})
	if _, err := term.AskHidden(ctx, "Password"); !errors.Is(err, context.Canceled) {
		t.Errorf("AskHidden = %v; want context.Canceled", err)
	}
}
