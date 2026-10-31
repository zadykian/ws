package wizard

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func TestSSHEnd(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/t/l"] = 0o700
	m.Files["/t/l/ws-ssh"] = nil
	m.Modes["/t/l/ws-ssh"] = fs.ModeSocket | 0o600
	m.Files["/t/l/root.err"] = nil
	m.Results["ssh -F /dev/null -O exit -o ControlPath=/t/l/ws-ssh ws"] = effecttest.Result{}
	SSH{M: m, Lock: "/t/l"}.End(context.Background())
	if got := strings.Join(m.Calls, "\n"); strings.Count(got, "run ssh") != 1 {
		t.Errorf("End calls %q; want one ssh -O exit", got)
	}
}
