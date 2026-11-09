package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// remoteLine is the ssh command line that Remote runs for the operation.
func remoteLine(r *Remote, op ...string) string {
	s := r.Shell.(SSHShell)
	args, _, _ := s.args(script(op), nil)
	return "ssh " + strings.Join(s.SSH.argv(context.Background(), args), " ")
}

func TestRemoteStatus(t *testing.T) {
	m := effecttest.NewMachine()
	r := &Remote{Shell: SSHShell{SSH: SSH{M: m, Lock: "/t/l"}}, Script: []byte("echo\n")}
	m.Results[remoteLine(r, "status")] = effecttest.Result{Out: []byte("run abc 0 off\nactive -\nclaude ok\n")}
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

func TestRemoteStatusFails(t *testing.T) {
	m := effecttest.NewMachine()
	r := &Remote{Shell: SSHShell{SSH: SSH{M: m, Lock: "/t/l"}}}
	m.Results[remoteLine(r, "status")] = effecttest.Result{Err: errors.New("denied")}
	if s, ok := r.status(context.Background()); ok || s != (wsStatus{}) {
		t.Error("status read from a failed ssh")
	}
}
