package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect"
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

// A mac.conf that is missing or refused says so, in place of ssh's want of a way.
func TestSSHWaySaysWhyMacConfLeavesNone(t *testing.T) {
	m := effecttest.NewMachine()
	s := SSH{M: m, Lock: "/l", Home: "/h", Conf: "/h/mac.conf"}
	if _, err := s.Way(context.Background()); err == nil || !strings.Contains(err.Error(), "no mac.conf at /h/mac.conf") {
		t.Errorf("Way with no mac.conf = %v", err)
	}
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=moon\n")
	if _, err := s.Way(context.Background()); err == nil || !strings.Contains(err.Error(), "/h/mac.conf, line 1") {
		t.Errorf("Way with a refused mac.conf = %v", err)
	}
	m.Root["/h/mac.conf"] = effect.Attrs{}
	if _, err := s.Way(context.Background()); err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("Way with an unreadable mac.conf = %v", err)
	}
}
