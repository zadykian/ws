package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
)

// blocked never answers, as a read of a terminal that nobody types at.
type blocked struct{}

func (blocked) Ask(context.Context, effect.Question) (string, error) {
	select {}
}

func TestSignalPromptEndsWithTheContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (SignalPrompt{blocked{}}).Ask(ctx, effect.Question{Text: "q"}); !errors.Is(err, context.Canceled) {
		t.Errorf("Ask after a signal = %v, want context.Canceled", err)
	}
}
