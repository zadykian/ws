package wizard

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestLockTakenOnce(t *testing.T) {
	m := effecttest.NewMachine()
	path, release, err := Lock(context.Background(), m, "/tmp/")
	if err != nil || path != "/tmp/ws-wizard.lock" {
		t.Fatalf("Lock = %q, %v", path, err)
	}
	if m.Dirs[path] != 0o700 {
		t.Errorf("lock mode = %o, want 700", m.Dirs[path])
	}
	_, _, err = Lock(context.Background(), m, "/tmp")
	if err == nil || !strings.Contains(err.Error(), "another run holds /tmp/ws-wizard.lock: wait for it to end") {
		t.Errorf("second Lock error = %v", err)
	}
	release()
	if !slices.Contains(m.Calls, "run rm -rf /tmp/ws-wizard.lock") {
		t.Errorf("release did not remove the lock: %v", m.Calls)
	}
}
