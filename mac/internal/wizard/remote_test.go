package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

// remoteLine is the ssh command line that Remote runs for the operation.
func remoteLine(r *Remote, op ...string) string {
	return "ssh " + strings.Join(r.SSH.argv(r.args(op)), " ")
}

func TestRemoteDo(t *testing.T) {
	m := effecttest.NewMachine()
	r := &Remote{SSH: SSH{M: m, Lock: "/t/l"}, Script: []byte("echo\n")}
	m.Results[remoteLine(r, "outbound")] = effecttest.Result{Out: []byte("done\n")}
	out, err := r.Do(context.Background(), "outbound")
	if err != nil || string(out) != "done\n" {
		t.Errorf("Do = %q, %v", out, err)
	}
}
