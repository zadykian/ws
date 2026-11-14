package remote

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestSSHWay(t *testing.T) {
	m := effecttest.NewMachine()
	s := SSH{Host: "ws", M: m, Lock: "/l", Home: "/h", Ways: fixedWays{host: "10.1.1.2", port: "22"}}
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
	s := SSH{Host: "ws", M: m, Lock: "/l", Home: "/h", Ways: fixedWays{}}
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

// The transport takes its direct way from Ways alone, and a Ways without one names its reason.
func TestSSHWayNamesTheReasonOfWays(t *testing.T) {
	reason := errors.New("no cable")
	s := SSH{Host: "ws", M: effecttest.NewMachine(), Ways: fixedWays{missing: reason}}
	if _, err := s.Way(context.Background()); !errors.Is(err, reason) {
		t.Errorf("Way = %v, want %v", err, reason)
	}
}

// wayCall is the ssh -G call that Way makes.
func wayCall(s SSH) string {
	return "ssh " + strings.Join(append([]string{"-G"}, s.Argv(context.Background(), []string{"ws"})...), " ")
}
