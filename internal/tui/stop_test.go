package tui

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestBridgeStop(t *testing.T) {
	b := newBridge(io.Discard)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	stop := b.Stop(ctx)
	if err := stop(); err != nil {
		t.Errorf("before a stop: %v", err)
	}
	cancel()
	if err := stop(); !errors.Is(err, context.Canceled) {
		t.Errorf("past ctx's end: %v", err)
	}
	b.halted.Store(true)
	if err := stop(); !errors.Is(err, errHalted) {
		t.Errorf("past a stop: %v", err)
	}
}
