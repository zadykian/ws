package systemd_test

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host/systemd"
)

func TestBooted(t *testing.T) {
	m := effecttest.NewMachine()
	if systemd.Booted(m) {
		t.Error("Booted without /run/systemd/system")
	}
	m.Dirs["/run/systemd/system"] = 0o755
	if !systemd.Booted(m) {
		t.Error("not Booted with /run/systemd/system")
	}
}
