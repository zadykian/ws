package main

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
)

// signalPrompt gives up a question when ctx ends. The run catches SIGINT, SIGTERM and SIGHUP into
// ctx, and a read of the terminal ignores it. So the read goes on in its own goroutine: the process
// ends with the run, as a trap did in sh, and takes the read with it.
type signalPrompt struct{ effect.Prompt }

type asked struct {
	answer string
	err    error
}

func (s signalPrompt) Ask(ctx context.Context, q effect.Question) (string, error) {
	done := make(chan asked, 1)
	go func() {
		answer, err := s.Prompt.Ask(ctx, q)
		done <- asked{answer, err}
	}()
	select {
	case got := <-done:
		return got.answer, got.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
