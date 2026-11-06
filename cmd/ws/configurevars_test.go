package main

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/wizard"
)

// lockCalls counts the calls of the wizard's lock that hold what.
func lockCalls(calls []string, what string) int {
	return len(slices.DeleteFunc(slices.Clone(calls), func(call string) bool {
		return !strings.Contains(call, what) || !strings.Contains(call, wizard.LockName)
	}))
}

func TestVarsReachTakesTheLockOnceAndLetsItGo(t *testing.T) {
	p, m := macTerminal()
	r := &varsReach{p: p}
	for range 2 {
		if _, err := r.reach(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "no direct way") {
			t.Fatalf("err = %v", err)
		}
	}
	if got := lockCalls(m.Calls, "mkdir"); got != 1 {
		t.Errorf("took the lock %d times", got)
	}
	r.close(context.Background())
	if got := lockCalls(m.Calls, "rm -rf"); got != 1 {
		t.Errorf("let it go %d times: %v", got, m.Calls)
	}
}

func TestVarsReachLeavesAnotherRunsLock(t *testing.T) {
	p, m := macTerminal()
	m.Dirs["/tmp/"+wizard.LockName] = 0o700
	r := &varsReach{p: p}
	if _, err := r.reach(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "another run holds") {
		t.Fatalf("err = %v", err)
	}
	r.close(context.Background())
	if got := lockCalls(m.Calls, "rm -rf"); got != 0 {
		t.Errorf("removed another run's lock: %v", m.Calls)
	}
}
