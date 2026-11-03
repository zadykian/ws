package jetbrains

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func exit(t *testing.T, code string) error {
	err := exec.Command("sh", "-c", "exit "+code).Run() //nolint:gosec // a test runs a shell that exits
	if err == nil {
		t.Fatal("no error")
	}
	return err
}

func TestRunningReadsPgrepsExitCode(t *testing.T) {
	m := effecttest.NewMachine()
	s := Site{M: m}
	m.Results["pgrep -f -- /d/a/"] = effecttest.Result{Out: []byte("12\n34\n")}
	m.Results["pgrep -f -- /d/b/"] = effecttest.Result{Err: exit(t, "1")}
	m.Results["pgrep -f -- /d/c/"] = effecttest.Result{Err: exit(t, "2")}
	if pids, err := s.running(context.Background(), "/d/a"); err != nil || len(pids) != 2 {
		t.Errorf("a: %v, %v", pids, err)
	}
	if pids, err := s.running(context.Background(), "/d/b"); err != nil || pids != nil {
		t.Errorf("b: %v, %v", pids, err)
	}
	if _, err := s.running(context.Background(), "/d/c"); err == nil {
		t.Error("c: a failed pgrep is no answer")
	}
}

func TestGetTriesFiveTimes(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["curl -fsSL https://x"] = effecttest.Result{Err: errors.New("down")}
	if _, err := (Site{M: m}).get(context.Background(), "https://x"); err == nil {
		t.Fatal("no error")
	}
	if len(m.Calls) != tries {
		t.Errorf("%d calls, want %d", len(m.Calls), tries)
	}
}
