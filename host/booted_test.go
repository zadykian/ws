package host_test

import (
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func TestBooted(t *testing.T) {
	m := effecttest.NewMachine()
	if host.Booted(m) {
		t.Error("Booted without /run/systemd/system")
	}
	m.Dirs["/run/systemd/system"] = 0o755
	if !host.Booted(m) {
		t.Error("not Booted with /run/systemd/system")
	}
}
