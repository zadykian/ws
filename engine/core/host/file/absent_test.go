package file_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host/file"
)

func TestAbsent(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/srv/old"] = []byte("x")
	m.Results["rm -rf -- /srv/old"] = effecttest.Result{}
	if got := applyAll(t, file.Absent{M: m, Path: "/srv/old"}); !slices.Equal(got, []string{"remove /srv/old"}) {
		t.Errorf("plan = %q, want the removal", got)
	}
	if !slices.Contains(m.Calls, "root rm -rf -- /srv/old") {
		t.Errorf("calls = %q, want rm as root", m.Calls)
	}
	if got := applyAll(t, file.Absent{M: m, Path: "/srv/never"}); got != nil {
		t.Errorf("plan of a missing path = %q, want none", got)
	}
}

func TestLineAbsent(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/f"] = []byte("a\nswap\nb\n/swapfile none swap sw 0 0\n")
	m.Modes["/f"] = 0o600
	l := file.Line{M: m, Path: "/f", Text: "swap", Match: `^/swapfile\b`, Absent: true}
	applyAll(t, l)
	if got := string(m.Files["/f"]); got != "a\nb\n" || m.Modes["/f"] != 0o600 {
		t.Errorf("file = %q, mode %o; want both lines gone, mode kept", got, m.Modes["/f"])
	}
	if again := applyAll(t, l); again != nil {
		t.Errorf("second plan = %q, want none", again)
	}
	if got := applyAll(t, file.Line{M: m, Path: "/missing", Text: "x", Absent: true}); got != nil {
		t.Errorf("plan of a missing file = %q, want none", got)
	}
}
