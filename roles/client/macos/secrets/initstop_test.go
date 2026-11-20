package secrets_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// stoppable is a Mac whose commands refuse to start once their context ends, as exec's do.
type stoppable struct{ *effecttest.Machine }

func (s stoppable) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.Machine.Run(ctx, name, args...)
}

// stopAt is init's Out, which stops the run once init prints the text: a Ctrl-C at that point.
type stopAt struct {
	strings.Builder
	text string
	stop context.CancelFunc
}

func (s *stopAt) Write(p []byte) (int, error) {
	if strings.Contains(string(p), s.text) {
		s.stop()
	}
	return s.Builder.Write(p)
}

func TestInitRemovesTheCloneWhenStoppedInTheWait(t *testing.T) {
	m := initMac(startTitle, startBody, "WS_GITHUB_TOKEN")
	m.Results[checksLine] = effecttest.Result{Out: []byte(oneRunning)}
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	i, _ := initOn(m, strings.NewReader("\n"), newBackup(t))
	i.M = stoppable{m}
	i.Sleep = func(ctx context.Context, _ time.Duration) error {
		stop()
		return ctx.Err()
	}
	if err := i.Run(ctx); err == nil || !strings.Contains(err.Error(), "stopped waiting for the checks") {
		t.Errorf("init stopped in the wait gives %v, want an error that it stopped waiting", err)
	}
	if m.Calls[len(m.Calls)-1] != "run rm -rf "+cloneBase {
		t.Errorf("init stopped in the wait leaves the clone: %q", m.Calls)
	}
}

func TestInitStopsAtTheBackupWhenStopped(t *testing.T) {
	m := initMac(startTitle, startBody, "WS_GITHUB_TOKEN")
	// A terminal where nobody presses Enter.
	in, w := io.Pipe()
	defer func() { _ = w.Close() }()
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	i, _ := initOn(m, in, newBackup(t))
	i.M, i.Out = stoppable{m}, &stopAt{text: "Press Enter", stop: stop}
	done := make(chan error, 1)
	go func() { done <- i.Run(ctx) }()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "nothing landed") {
			t.Errorf("init stopped at the backup gives %v, want an error that nothing landed", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("init stopped at the backup still waits for Enter")
	}
	if count(m.Calls, "run "+pushLine) != 0 || m.Calls[len(m.Calls)-1] != "run rm -rf "+cloneBase {
		t.Errorf("init stopped at the backup pushed, or left the clone: %q", m.Calls)
	}
}
