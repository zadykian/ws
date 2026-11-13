package remote

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
	m.Files["/t/l/lab"] = nil
	m.Modes["/t/l/lab"] = fs.ModeSocket | 0o600
	m.Results["ssh -F /dev/null -O exit -o ControlPath=/t/l/direct direct"] = effecttest.Result{}
	m.Results["ssh -F /dev/null -O exit -o ControlPath=/t/l/lab lab"] = effecttest.Result{}
	SSH{Host: "ws", M: m, Lock: "/t/l"}.End(context.Background())
	if got := strings.Join(m.Calls, "\n"); strings.Count(got, "run ssh") != 2 {
		t.Errorf("End calls %q; want an ssh -O exit for each socket", got)
	}
	for _, sock := range []string{"direct direct", "lab lab"} {
		if got := strings.Join(m.Calls, "\n"); !strings.Contains(got, "/t/l/"+sock) {
			t.Errorf("End calls %q; want %q", got, sock)
		}
	}
}
