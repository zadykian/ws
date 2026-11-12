package host_test

import (
	"context"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func TestLookupFindsTheAccount(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1001\n")}
	a, err := host.User{Name: "ws", Home: "/home/ws"}.Lookup(context.Background(), m)
	if err != nil || a.UID != 1000 || a.GID != 1001 || a.Home != "/home/ws" {
		t.Errorf("got %+v, %v", a, err)
	}
	root, err := host.User{Name: "root", Home: "/root"}.Lookup(context.Background(), m)
	if err != nil || root.UID != 0 || root.GID != 0 || len(m.Calls) != 2 {
		t.Errorf("root: %+v, %v, calls %q", root, err, m.Calls)
	}
	if _, err := (host.User{Name: "nobody-here"}).Lookup(context.Background(), m); err == nil {
		t.Error("an unknown user is found")
	}
}
