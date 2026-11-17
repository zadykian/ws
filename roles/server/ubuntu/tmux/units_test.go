package tmux_test

import (
	"errors"
	"os"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// asRoot is how a run state runs a command as root: in /root, with the system's PATH.
const asRoot = "env HOME=/root env XDG_RUNTIME_DIR=/run/user/0 " +
	"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/0/bus " +
	"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin "
const (
	check  = asRoot + "sh -c [ ! -x /usr/bin/snap ] || ! snap list tmux >/dev/null 2>&1"
	remove = asRoot + "snap remove tmux"
)

// snapUnit loads the role's unit on a machine with systemd or without. The fake machine runs no
// shell, so held sets the check's answer.
func snapUnit(t *testing.T, booted, held bool) (*effecttest.Machine, *effecttest.Out, kind.Unit) {
	text, err := os.ReadFile("units.toml")
	if err != nil {
		t.Fatal(err)
	}
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	if booted {
		m.Dirs["/run/systemd/system"] = 0o755
	}
	m.Results[check], m.Results[remove] = effecttest.Result{}, effecttest.Result{}
	if held {
		m.Results[check] = effecttest.Result{Err: errors.New("exit status 1")}
	}
	e := kind.Env{M: m, U: &effecttest.Units{}, Out: out, Data: v}
	us, err := kind.Load(e, fstest.MapFS{"tmux/units.toml": &fstest.MapFile{Data: text}})
	if err != nil || len(us) != 1 || us[0].Name != "state/tmux-snap" {
		t.Fatalf("units = %v, %v; want state/tmux-snap alone", kind.Names(us), err)
	}
	return m, out, us[0]
}

func plan(t *testing.T, u kind.Unit) step.Plan {
	p, err := u.Step.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestTmuxSnapRemovedOnce(t *testing.T) {
	m, _, u := snapUnit(t, true, true)
	p := plan(t, u)
	if len(p.Changes) != 1 || step.Headline(p.Changes[0]) != "run snap remove tmux as root" {
		t.Fatalf("plan with the snap = %v, want its removal", p.Changes)
	}
	if err := u.Step.Apply(t.Context(), p); err != nil || !slices.Contains(m.Calls, "run "+remove) {
		t.Errorf("apply = %v, calls %q; want snap remove tmux as root", err, m.Calls)
	}
	m.Results[check] = effecttest.Result{}
	if p := plan(t, u); len(p.Changes) != 0 {
		t.Errorf("second plan = %v, want none", p.Changes)
	}
}

func TestTmuxSnapSkipsWithoutSystemd(t *testing.T) {
	m, out, u := snapUnit(t, false, true)
	if p := plan(t, u); len(p.Changes) != 0 || slices.Contains(m.Calls, "run "+check) || len(out.Lines) != 1 {
		t.Errorf("plan = %v, calls %q, said %q; want nothing, no check, one skip said", p.Changes, m.Calls, out.Lines)
	}
}

// The check passes without snapd, as it does without the snap: its first branch is the guard.
func TestTmuxSnapWithoutSnapdPlansNothing(t *testing.T) {
	_, _, u := snapUnit(t, true, false)
	if p := plan(t, u); len(p.Changes) != 0 {
		t.Errorf("plan = %v, want none", p.Changes)
	}
}
