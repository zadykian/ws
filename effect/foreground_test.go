package effect_test

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/zadykian/ws/effect"
)

var _ effect.Foreground = effect.OSMachine{}

// shown puts a pipe in place of os.Stderr, and returns what reads it.
func shown(t *testing.T) *bufio.Reader {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	t.Cleanup(func() {
		os.Stderr = old
		_ = r.Close()
		_ = w.Close()
	})
	return bufio.NewReader(r)
}

// wantLine fails the test unless the next line is want.
func wantLine(t *testing.T, r *bufio.Reader, want string) {
	t.Helper()
	if line, err := r.ReadString('\n'); err != nil || line != want+"\n" {
		t.Fatalf("line = %q, %v; want %q", line, err, want)
	}
}

func TestOSMachinePassForeground(t *testing.T) {
	t.Setenv("GIT_TERMINAL_PROMPT", "1")
	typed(t, "typed\n")
	terminal := shown(t)
	var pane bytes.Buffer
	m := effect.OSMachine{Sudo: []string{"env", "WS_TEST=1"}, Stdin: effect.NoStdin, Stderr: &pane}
	err := m.PassForeground(t.Context(), true, "sh", "-c",
		`read -r line; echo "$line $WS_TEST $GIT_TERMINAL_PROMPT"; echo err >&2`)
	if err != nil {
		t.Fatal(err)
	}
	wantLine(t, terminal, "typed 1 1")
	wantLine(t, terminal, "err")
	if pane.Len() != 0 {
		t.Errorf("Stderr took %q, want nothing", pane.String())
	}
	if err := m.PassForeground(t.Context(), false, "false"); err == nil {
		t.Error("a failing command: no error")
	}
}

// ended runs script in the foreground, ends its context once it prints ready, and returns what
// reads the rest of its output, and its error. The pipe closes as the command ends, so that a read
// past it fails where a line is missing.
func ended(t *testing.T, script string) (*bufio.Reader, error) {
	t.Helper()
	terminal := shown(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		err := (effect.OSMachine{}).PassForeground(ctx, false, "sh", "-c", script)
		_ = os.Stderr.Close()
		done <- err
	}()
	wantLine(t, terminal, "ready")
	cancel()
	return terminal, <-done
}

func TestOSMachinePassForegroundEndsByTerm(t *testing.T) {
	terminal, err := ended(t, `trap 'echo term >&2; exit 3' TERM; echo ready >&2; while :; do sleep 0.01; done`)
	if err == nil {
		t.Error("an ended command: no error")
	}
	wantLine(t, terminal, "term")
}

func TestOSMachinePassForegroundKillsPastTheGrace(t *testing.T) {
	if _, err := ended(t, `trap '' TERM; echo ready >&2; while :; do sleep 0.01; done`); err == nil {
		t.Error("a killed command: no error")
	}
}
