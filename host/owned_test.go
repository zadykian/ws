package host_test

import (
	"context"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func TestUserIDs(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1001\n")}
	uid, gid, err := host.User{Name: "ws"}.IDs(context.Background(), m)
	if err != nil || uid != 1000 || gid != 1001 {
		t.Errorf("got %d %d, %v", uid, gid, err)
	}
	uid, gid, err = host.User{Name: "root"}.IDs(context.Background(), m)
	if err != nil || uid != 0 || gid != 0 || len(m.Calls) != 2 {
		t.Errorf("root: %d %d, %v, calls %q", uid, gid, err, m.Calls)
	}
	if _, _, err := (host.User{Name: "nobody-here"}).IDs(context.Background(), m); err == nil {
		t.Error("an unknown user is found")
	}
}

func TestOwnedMakesItsPartsWithTheIDs(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1001\n")}
	s := host.User{Name: "ws"}.Owned(m, "Files", "", func(uid, gid int) ([]host.Resource, error) {
		return []host.Resource{host.Dir{M: m, Path: "/home/ws/d", Mode: 0o755, UID: uid, GID: gid}}, nil
	})
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 1 {
		t.Fatalf("plan %v, %v", p, err)
	}
	if got, want := p.Changes[0].String(), "create directory /home/ws/d, owner 1000:1001, mode 755"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
