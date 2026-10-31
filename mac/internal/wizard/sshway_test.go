package wizard

import (
	"context"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func TestSSHWay(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["ssh -G ws-ssh"] = effecttest.Result{Out: []byte(
		"user root\nhostname 10.1.1.2\nport 22\nuserknownhostsfile ~/.ssh/known_hosts ~/x\naddressfamily inet\n")}
	w, err := SSH{M: m, Home: "/h"}.Way(context.Background())
	want := Way{Host: "10.1.1.2", Port: "22", User: "root", KnownHosts: "/h/.ssh/known_hosts", Inet: true}
	if err != nil || w != want {
		t.Errorf("Way = %+v, %v; want %+v", w, err, want)
	}
	m.Results["ssh -G ws-ssh"] = effecttest.Result{Out: []byte("hostname ws-ssh\n")}
	if _, err := (SSH{M: m}).Way(context.Background()); err == nil {
		t.Error("Way of ws-ssh as its own host succeeded")
	}
}
