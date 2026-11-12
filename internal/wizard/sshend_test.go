package wizard

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestSSHEnd(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/t/l"] = 0o700
	m.Files["/t/l/direct"] = nil
	m.Modes["/t/l/direct"] = fs.ModeSocket | 0o600
	m.Files["/t/l/root.err"] = nil
	m.Results["ssh -F /dev/null -O exit -o ControlPath=/t/l/direct ws"] = effecttest.Result{}
	SSH{M: m, Lock: "/t/l"}.End(context.Background())
	if got := strings.Join(m.Calls, "\n"); strings.Count(got, "run ssh") != 1 {
		t.Errorf("End calls %q; want one ssh -O exit", got)
	}
}
