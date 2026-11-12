package host_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
)

const (
	userShow = "root runuser -u ws -- env -C /home/ws HOME=/home/ws env XDG_RUNTIME_DIR=/run/user/1000 " +
		"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus systemctl --user "
	userShowState = "show --property=ActiveState --value docker.service"
)

func userUnitFixture(booted bool) (*effecttest.Machine, host.UserUnit, *effecttest.Out) {
	m := effecttest.NewMachine()
	if booted {
		m.Dirs["/run/systemd/system"] = 0o755
	}
	for _, c := range []string{"daemon-reload", "start docker.service"} {
		m.Results[userShow[len("root "):]+c] = effecttest.Result{}
	}
	out := &effecttest.Out{}
	return m, host.UserUnit{M: m, Account: host.Account{User: host.User{Name: "ws", Home: "/home/ws"}, UID: 1000}, Out: out,
		Name: "docker.service"}, out
}

func userState(m *effecttest.Machine, state string) {
	m.Dirs["/run/user/1000/systemd/private"] = 0o755
	m.Results[userShow[len("root "):]+userShowState] = effecttest.Result{Out: []byte(state + "\n")}
}

func TestUserUnitSkipsWithoutSystemd(t *testing.T) {
	_, u, out := userUnitFixture(false)
	if got := applyAll(t, u); got != nil || len(out.Lines) != 1 {
		t.Errorf("plan = %q, said %q; want a skip said once", got, out.Lines)
	}
}

func TestUserUnitWithoutAFileIsLeftAlone(t *testing.T) {
	_, u, _ := userUnitFixture(true)
	if got := applyAll(t, u); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
}

func TestUserUnitStartsWhereTheManagerIsDown(t *testing.T) {
	m, u, _ := userUnitFixture(true)
	m.Files["/home/ws/.config/systemd/user/docker.service"] = nil
	if got := applyAll(t, u); !slices.Equal(got, []string{"start ws's docker.service"}) {
		t.Errorf("plan = %q, want a start", got)
	}
}

func TestUserUnitStartsAnInactiveUnitAndLeavesAnActiveOne(t *testing.T) {
	m, u, _ := userUnitFixture(true)
	m.Files["/home/ws/.config/systemd/user/docker.service"] = nil
	userState(m, "inactive")
	if got := applyAll(t, u); !slices.Equal(got, []string{"start ws's docker.service"}) {
		t.Fatalf("plan = %q, want a start", got)
	}
	if !slices.Contains(m.Calls, "root runuser -u ws -- env -C /home/ws HOME=/home/ws env XDG_RUNTIME_DIR=/run/user/1000 "+
		"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus systemctl --user start docker.service") {
		t.Errorf("calls = %q, want systemctl --user start", m.Calls)
	}
	userState(m, "active")
	if got := applyAll(t, u); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
}

func TestUserUnitRestartsAfterAWatchedChange(t *testing.T) {
	m, u, out := userUnitFixture(true)
	m.Files["/home/ws/.config/systemd/user/docker.service"] = nil
	m.Results[userShow[len("root "):]+"restart docker.service"] = effecttest.Result{}
	userState(m, "active")
	conf := host.File{M: m, Path: "/home/ws/.config/docker/daemon.json", Content: []byte("x"), Mode: 0o644}
	u.Watch = []host.Resource{conf}
	g := host.Group{Title: "A", Parts: []host.Resource{conf, u}}
	if _, err := step.Run(t.Context(), []step.Step{g}, out, false); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Calls, userShow+"restart docker.service") {
		t.Errorf("calls = %q, want a restart", m.Calls)
	}
	if _, ok := m.Files["/var/lib/ws/owed/ws-docker.service"]; ok {
		t.Error("the mark is left after the restart")
	}
	if got := applyAll(t, u); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
}

func TestUserUnitRestartsWhereAMarkIsLeft(t *testing.T) {
	m, u, _ := userUnitFixture(true)
	m.Files["/home/ws/.config/systemd/user/docker.service"] = nil
	m.Results[userShow[len("root "):]+"restart docker.service"] = effecttest.Result{}
	m.Root["/var/lib/ws/owed/ws-docker.service"] = effect.Attrs{}
	m.Files["/var/lib/ws/owed/ws-docker.service"] = nil
	userState(m, "active")
	if got := applyAll(t, u); !slices.Equal(got, []string{"restart ws's docker.service"}) {
		t.Errorf("plan = %q, want the restart that the mark owes", got)
	}
}

func TestUserUnitReport(t *testing.T) {
	_, u, _ := userUnitFixture(false)
	if got := reportOf(t, u); got.Name != "ws's docker.service" || got.Found != "systemd is not running" {
		t.Errorf("item = %+v, want the user's unit and the state of a machine with no systemd", got)
	}
	m, u, _ := userUnitFixture(true)
	if got := reportOf(t, u); got.Found != "stopped" {
		t.Errorf("item = %+v, want stopped with the manager down", got)
	}
	userState(m, "active")
	if got := reportOf(t, u); got.Want != "running" || got.Found != "running" {
		t.Errorf("item = %+v, want it running", got)
	}
}
