package wizard

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func TestSSHArgv(t *testing.T) {
	for name, tc := range map[string]struct {
		lock, first, want string
		notWant           string
	}{
		"shared":  {"/t/l", "", "ControlPath=/t/l/%n ", "HostName"},
		"first":   {"/t/l", "10.0.0.5", "ControlPath=/t/l/%n.1 -o ControlPersist=30m", ""},
		"no room": {"/t/%l", "", "ConnectTimeout=10", "ControlMaster"},
	} {
		m := effecttest.NewMachine()
		if tc.first != "" {
			m.Files[tc.lock+"/"+FirstFile] = []byte(tc.first + "\n")
		}
		got := strings.Join(SSH{M: m, Lock: tc.lock}.argv([]string{"ws-ssh"}), " ") + " "
		if !strings.Contains(got, tc.want) || (tc.notWant != "" && strings.Contains(got, tc.notWant)) {
			t.Errorf("%s: argv = %q; want %q, not %q", name, got, tc.want, tc.notWant)
		}
	}
}
