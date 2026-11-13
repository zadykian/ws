package tui

import (
	"context"
	"errors"
	"io"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/tui/internal/run"
)

// typedAtRead returns a terminal, and a read that types the next of keys on it, again and again
// until the terminal closes. `r` finds its button off while a read runs, and goes unheard there.
func typedAtRead(t *testing.T, keys ...string) (Config, StateReader) {
	in, typing := io.Pipe()
	t.Cleanup(func() { _ = typing.Close() })
	c := noKeys(io.Discard)
	c.In = in
	return c, func(context.Context, *Bridge) ([]StateSection, error) {
		if len(keys) == 0 {
			return testSections(), nil
		}
		next := []byte(keys[0])
		keys = keys[1:]
		go func() {
			for {
				if _, err := typing.Write(next); err != nil {
					return
				}
				time.Sleep(time.Millisecond)
			}
		}()
		return testSections(), nil
	}
}

func TestRunStateReadsAtRAndEndsAtQ(t *testing.T) {
	c, read := typedAtRead(t, "r", "q")
	if err := RunState(t.Context(), c, "ws", read); err != nil {
		t.Errorf("RunState: %v", err)
	}
}

func TestRunStateEndsARunningReadAtQ(t *testing.T) {
	c, read := typedAtRead(t, "q")
	err := RunState(t.Context(), c, "ws", func(ctx context.Context, b *Bridge) ([]StateSection, error) {
		_, _ = read(ctx, b)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	if err != nil {
		t.Errorf("RunState: %v", err)
	}
}

func TestRunStateGivesTheCutsError(t *testing.T) {
	c, read := typedAtRead(t, "\x03")
	if err := RunState(t.Context(), c, "ws", read); !errors.Is(err, testCut(syscall.SIGINT)) {
		t.Errorf("RunState: %v", err)
	}
}

func TestRunStateTurnsAPanicOfAReadIntoAnError(t *testing.T) {
	var out strings.Builder
	err := RunState(t.Context(), noKeys(&out), "ws", func(context.Context, *Bridge) ([]StateSection, error) {
		panic("a bug")
	})
	if !errors.Is(err, run.ErrPanic) || !strings.Contains(out.String(), leaveBuffer) {
		t.Errorf("RunState: %v, on %q", err, out.String())
	}
}
