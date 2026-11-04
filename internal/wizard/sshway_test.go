package wizard

import (
	"context"
	"errors"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestSSHWay(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\n" + wayConf)
	s := SSH{M: m, Lock: "/l", Home: "/h", Conf: "/h/mac.conf"}
	m.Results[wayCall(s)] = effecttest.Result{Out: []byte(
		"user root\nhostname 10.1.1.2\nport 22\nuserknownhostsfile ~/.ssh/known_hosts ~/x\naddressfamily inet\n")}
	w, err := s.Way(context.Background())
	want := Way{Host: "10.1.1.2", Port: "22", User: "root", KnownHosts: "/h/.ssh/known_hosts", Inet: true}
	if err != nil || w != want {
		t.Errorf("Way = %+v, %v; want %+v", w, err, want)
	}
}

func TestSSHWayNeedsAWayOrFirst(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\n")
	s := SSH{M: m, Lock: "/l", Home: "/h", Conf: "/h/mac.conf"}
	if _, err := s.Way(context.Background()); !errors.Is(err, errNoWay) {
		t.Errorf("Way with no way and no first = %v; want errNoWay", err)
	}
	m.Files["/l/"+FirstFile] = []byte("10.1.1.9\n")
	m.Results[wayCall(s)] = effecttest.Result{Out: []byte("hostname 10.1.1.9\n")}
	if w, err := s.Way(context.Background()); err != nil || w.Host != "10.1.1.9" {
		t.Errorf("Way with first alone = %+v, %v; want 10.1.1.9", w, err)
	}
	m.Results[wayCall(s)] = effecttest.Result{Err: errNoWay}
	if _, err := s.Way(context.Background()); !errors.Is(err, errNoWay) {
		t.Errorf("Way where ssh -G fails = %v; want errNoWay", err)
	}
}
