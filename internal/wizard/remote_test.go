package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestRemoteStatus(t *testing.T) {
	m := effecttest.NewMachine()
	r := &Remote{Shell: fakeWs{m}, Script: []byte("echo\n")}
	m.Results[opLine("status")] = effecttest.Result{Out: []byte("run abc 0 off\nactive -\nclaude ok\n")}
	s, ok := r.status(context.Background())
	if !ok {
		t.Fatal("status read nothing")
	}
	if s.Run.Exit != "" || s.Claude != "ok" {
		t.Errorf("status = %+v", s)
	}
	r.status(context.Background())
	n := 0
	for _, c := range m.Calls {
		if strings.Contains(c, "status") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("status ran %d times; want 1", n)
	}
}

// A change through the files script has the status read ws again.
func TestRemoteBesideLeavesStatusStale(t *testing.T) {
	m := effecttest.NewMachine()
	r := &Remote{Shell: fakeWs{m}, Script: []byte("echo\n")}
	files := r.Beside([]byte("true\n"))
	m.Results[opLine("status")] = effecttest.Result{Out: []byte("cable -\n")}
	r.status(context.Background())
	m.Results[opLine("link", "enp1s0")] = effecttest.Result{}
	if _, err := files.Do(context.Background(), "link", "enp1s0"); err != nil {
		t.Fatal(err)
	}
	m.Results[opLine("status")] = effecttest.Result{Out: []byte("cable enp1s0\n")}
	if s, _ := r.status(context.Background()); s.Cable != "enp1s0" {
		t.Errorf("cable = %q; want the status read again", s.Cable)
	}
}

func TestRemoteStatusFails(t *testing.T) {
	m := effecttest.NewMachine()
	r := &Remote{Shell: fakeWs{m}}
	m.Results[opLine("status")] = effecttest.Result{Err: errors.New("denied")}
	if s, ok := r.status(context.Background()); ok || s != (wsStatus{}) {
		t.Error("status read from a failed ssh")
	}
}
