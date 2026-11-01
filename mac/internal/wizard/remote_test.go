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
	return "ssh " + strings.Join(r.SSH.argv(r.args(op)), " ")
}

func TestRemoteStatus(t *testing.T) {
	m := effecttest.NewMachine()
	r := &Remote{SSH: SSH{M: m, Lock: "/t/l"}, Script: []byte("echo\n")}
	m.Results[remoteLine(r, "status")] = effecttest.Result{Out: []byte("run abc 0 off\nactive -\nclaude ok\n")}
	if !r.Status(context.Background()) {
		t.Fatal("Status read nothing")
	}
	if got := r.Get("run"); got != "abc 0 off" {
		t.Errorf("run = %q", got)
	}
	r.Status(context.Background())
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
	r := &Remote{SSH: SSH{M: m, Lock: "/t/l"}}
	m.Results[remoteLine(r, "status")] = effecttest.Result{Err: errors.New("denied")}
	if r.Status(context.Background()) || r.Get("run") != "" {
		t.Error("Status read from a failed ssh")
	}
}
