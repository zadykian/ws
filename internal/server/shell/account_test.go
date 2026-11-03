package shell

import (
	"context"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func TestLookup(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1001\n")}
	a, err := lookup(context.Background(), m, host.User{Name: "ws", Home: "/home/ws"})
	if err != nil || a.UID != 1000 || a.GID != 1001 {
		t.Errorf("got %+v, %v", a, err)
	}
	root, err := lookup(context.Background(), m, host.User{Name: "root", Home: "/root"})
	if err != nil || root.UID != 0 || len(m.Calls) != 2 {
		t.Errorf("root: %+v, %v, calls %q", root, err, m.Calls)
	}
	if _, err := lookup(context.Background(), m, host.User{Name: "nobody-here"}); err == nil {
		t.Error("an unknown user is found")
	}
}
