package account_test

import (
	"context"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/account"
)

func TestLookupFindsTheAccount(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1001\n")}
	a, err := account.User{Name: "ws", Home: "/home/ws"}.Lookup(context.Background(), m)
	if err != nil || a.UID != 1000 || a.GID != 1001 || a.Home != "/home/ws" {
		t.Errorf("got %+v, %v", a, err)
	}
	root, err := account.User{Name: "root", Home: "/root"}.Lookup(context.Background(), m)
	if err != nil || root.UID != 0 || root.GID != 0 || len(m.Calls) != 2 {
		t.Errorf("root: %+v, %v, calls %q", root, err, m.Calls)
	}
	if _, err := (account.User{Name: "nobody-here"}).Lookup(context.Background(), m); err == nil {
		t.Error("an unknown user is found")
	}
}

func TestWithAccountMakesItsPartsFromTheAccount(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1001\n")}
	s := account.User{Name: "ws", Home: "/home/ws"}.WithAccount(m, "Files", "", func(a account.Account) ([]host.Resource, error) {
		return []host.Resource{host.Dir{M: m, Path: a.Home + "/d", Mode: 0o755, UID: a.UID, GID: a.GID}}, nil
	})
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 1 {
		t.Fatalf("plan %v, %v", p, err)
	}
	if got, want := p.Changes[0].String(), "create directory /home/ws/d, owner 1000:1001, mode 755"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
