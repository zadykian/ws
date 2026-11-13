package cli

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// fakeProcess returns an Env that touches nothing of the test's own.
func fakeProcess() enginecli.Env { return clitest.Env() }

// runWs runs args on the root of ws server, with stdin, and returns the status, stdout and stderr.
func runWs(p enginecli.Env, stdin string, args ...string) (status int, stdout, stderr string) {
	return clitest.Run(p, []*cobra.Command{serverGroup(p)}, stdin, args...)
}

// serverGroup is ws server, over this package's commands.
func serverGroup(p enginecli.Env) *cobra.Command {
	return enginecli.Group("server", "the server's systems", Command(p))
}

// withEnv returns p with the environment env, for Getenv, Lookup and Setenv.
func withEnv(p enginecli.Env, env map[string]string) enginecli.Env { return clitest.WithEnv(p, env) }

const (
	// enterScreen and leaveScreen are how a terminal's alternate screen starts and ends.
	enterScreen = "\x1b[?1049h"
	leaveScreen = "\x1b[?1049l"
)

// repeating types keys again and again, so that one lands once the list takes it.
type repeating string

func (r repeating) Read(b []byte) (int, error) {
	time.Sleep(5 * time.Millisecond)
	return copy(b, r), nil
}

// afterScan is a terminal that holds keys back until the list draws its count in place of the scan's.
// Keys of one read can straddle the scan's end, and those past it then act on a list that missed the
// first. It reports a size first, as a screen with none draws nothing. A list that never draws the
// count gets the keys all the same, and the test fails on what they did.
type afterScan struct {
	keys  repeating
	sized sync.Once
	mu    sync.Mutex
	drawn bytes.Buffer
	shown sync.Once
	ended chan struct{}
}

func newAfterScan(keys repeating) *afterScan {
	return &afterScan{keys: keys, ended: make(chan struct{})}
}

func (s *afterScan) Write(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.drawn.Write(b)
	if bytes.Contains(s.drawn.Bytes(), []byte(" chosen")) {
		s.shown.Do(func() { close(s.ended) })
	}
	return len(b), nil
}

func (s *afterScan) Read(b []byte) (int, error) {
	size := 0
	s.sized.Do(func() { size = copy(b, "\x1b[8;24;80t") })
	if size > 0 {
		return size, nil
	}
	select {
	case <-s.ended:
	case <-time.After(10 * time.Second):
	}
	return s.keys.Read(b)
}

// freed is a machine whose rm frees the lock, for the run that a scan's end lets in.
type freed struct{ *effecttest.Machine }

func (f freed) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "rm" {
		delete(f.Dirs, args[len(args)-1])
	}
	return f.Machine.Run(ctx, name, args...)
}

// stateTrip is a terminal that presses s on the lists, q on the Mac's state screen, and q on the lists
// that follow. The keys go by the screen drawn, as a reader of a program that ended takes one.
// stateTrip leaves the state screen titled title with q.
type stateTrip struct {
	*afterScan
	title string
}

func (s stateTrip) Read(b []byte) (int, error) {
	n, err := s.afterScan.Read(b)
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case bytes.Contains(s.drawn.Bytes(), []byte(s.title)):
		return copy(b, "q"), nil
	case bytes.Count(s.drawn.Bytes(), []byte(enterScreen)) > 1:
		return copy(b, "\x1b[8;24;80t"), nil // A later screen draws from its size too.
	}
	return n, err
}
