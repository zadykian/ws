package host_test

import (
	"context"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func TestLingerPlan(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	l := host.Linger{M: m, User: host.User{Name: "ws"}, Out: &effecttest.Out{}}
	if got, err := l.Plan(context.Background()); err != nil || len(got) != 1 {
		t.Fatalf("%v, %v", got, err)
	}
	m.Files["/var/lib/systemd/linger/ws"] = nil
	if got, err := l.Plan(context.Background()); err != nil || len(got) != 0 {
		t.Fatalf("%v, %v", got, err)
	}
}

func TestLingerSkipsWithoutSystemd(t *testing.T) {
	m := effecttest.NewMachine()
	l := host.Linger{M: m, User: host.User{Name: "ws"}, Out: &effecttest.Out{}}
	if got, err := l.Plan(context.Background()); err != nil || len(got) != 0 {
		t.Fatalf("%v, %v", got, err)
	}
}
