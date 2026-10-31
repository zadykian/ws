package effect_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect"
)

func TestOSMachineRun(t *testing.T) {
	ctx := context.Background()
	m := effect.OSMachine{Sudo: []string{"env", "WS_TEST=1"}}
	if out, err := m.Run(ctx, "echo", "hi"); err != nil || string(out) != "hi\n" {
		t.Errorf("Run = %q, %v; want hi", out, err)
	}
	if out, err := m.RunRoot(ctx, "sh", "-c", "echo $WS_TEST"); err != nil || string(out) != "1\n" {
		t.Errorf("RunRoot = %q, %v; want 1", out, err)
	}
	_, err := m.Run(ctx, "sh", "-c", "echo boom >&2; exit 3")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("Run of a failing command: err = %v, want its stderr", err)
	}
}
