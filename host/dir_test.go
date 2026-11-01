package host_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func TestDir(t *testing.T) {
	m := effecttest.NewMachine()
	d := host.Dir{M: m, Path: "/srv/d", Mode: 0o750, UID: 1000, GID: 1000}
	if got := applyAll(t, d); len(got) != 1 || !strings.HasPrefix(got[0], "create directory /srv/d") {
		t.Errorf("first plan = %q, want a create", got)
	}
	if got := applyAll(t, d); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
	d.Mode = 0o700
	if got := applyAll(t, d); len(got) != 1 || !strings.HasPrefix(got[0], "set owner 1000:1000 and mode 700") {
		t.Errorf("mode plan = %q", got)
	}
	if m.Dirs["/srv/d"] != 0o700 {
		t.Errorf("mode = %o, want 700", m.Dirs["/srv/d"])
	}
}
