package launchd_test

import (
	"testing"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
)

func TestLaunchctlSaysItsKind(t *testing.T) {
	if got := step.KindOf(launchd.Launchctl{}); got != step.Restart {
		t.Errorf("kind = %s, want %s", got, step.Restart)
	}
}
