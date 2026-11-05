package docker

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func installFixture(booted bool) (*effecttest.Machine, rootlessInstall, *effecttest.Out) {
	m := effecttest.NewMachine()
	if booted {
		m.Dirs["/run/systemd/system"] = 0o755
	}
	out := &effecttest.Out{}
	return m, rootlessInstall{m, host.User{Name: "ws", Home: "/home/ws"}, 1000, out}, out
}

func TestRootlessInstallRunsUntilTheUnitAndItsLinkExist(t *testing.T) {
	m, r, _ := installFixture(true)
	changes, err := r.Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Fatalf("Plan = %v, %v; want the install", changes, err)
	}
	units := "/home/ws/.config/systemd/user"
	m.Files[units+"/docker.service"] = []byte("[Unit]\n")
	if changes, _ := r.Plan(t.Context()); len(changes) != 1 {
		t.Errorf("a unit with no link plans %v; want the install again", changes)
	}
	m.Files[units+"/default.target.wants/docker.service"] = []byte("[Unit]\n")
	if changes, err := r.Plan(t.Context()); err != nil || changes != nil {
		t.Errorf("Plan = %v, %v; want none once the unit is linked", changes, err)
	}
}

func TestRootlessInstallRunsDockersTool(t *testing.T) {
	m, r, _ := installFixture(true)
	session := "runuser -u ws -- env -C /home/ws HOME=/home/ws env XDG_RUNTIME_DIR=/run/user/1000 " +
		"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus "
	m.Results[session+"dockerd-rootless-setuptool.sh install"] = effecttest.Result{}
	if err := r.Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	calls := strings.Join(m.Calls, "\n")
	if !strings.Contains(calls, "dockerd-rootless-setuptool.sh install") || !strings.Contains(calls, "XDG_RUNTIME_DIR=/run/user/1000") {
		t.Errorf("calls = %q; want the tool, in the user's session", calls)
	}
}
