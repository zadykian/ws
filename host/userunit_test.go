package host_test

import (
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func userUnitFixture(booted bool) (*effecttest.Machine, host.UserUnit, *effecttest.Out) {
	m := effecttest.NewMachine()
	if booted {
		m.Dirs["/run/systemd/system"] = 0o755
	}
	out := &effecttest.Out{}
	return m, host.UserUnit{M: m, User: host.User{Name: "ws", Home: "/home/ws"}, UID: 1000, Out: out,
		Name: "docker.service"}, out
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
