package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// badSiblingFixture is ws beside lab, whose file is refused for a slot out of range, and the
// resolver file that lab's run made.
func badSiblingFixture() (*effecttest.Machine, string) {
	m := effecttest.NewMachine()
	keyed(m)
	m.Files["/h/.config/ws/servers/lab/server.conf"] = []byte("WS_USER=ada\nWS_SLOT=99\n")
	m.Dirs["/h/.config/ws/servers/lab"] = 0o755
	m.Dirs["/etc/resolver"] = 0o755
	resolver := "/etc/resolver/lab.test"
	m.Files[resolver] = []byte("# ws\nnameserver 10.30.0.53\n")
	return m, resolver
}

// TestApplyRefusesABadSiblingBeforeAnyStep settles R1071-3. The networks step prunes the resolver
// files of a server it cannot read. So a run for ws stops where lab's file is refused, with --only
// and with --check too.
func TestApplyRefusesABadSiblingBeforeAnyStep(t *testing.T) {
	for _, args := range [][]string{
		{"client", "macos", "apply", "--server", "ws"},
		{"client", "macos", "apply", "--server", "ws", "--only", "networks"},
		{"client", "macos", "apply", "--server", "ws", "--only", "networks", "--check"},
	} {
		m, resolver := badSiblingFixture()
		p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
		p.Machine = m
		status, _, stderr := runWs(p, "", args...)
		if status != 1 || !strings.Contains(stderr, "server lab: ") || !strings.Contains(stderr, "fix its file, or ws client macos remove lab") {
			t.Errorf("%q: status = %d, stderr = %q; want lab's file refused, naming remove", args, status, stderr)
		}
		for _, call := range m.Calls {
			if !slices.Contains([]string{"read", "readdir", "stat"}, strings.Fields(call)[0]) {
				t.Errorf("%q: call %q ran before the refusal", args, call)
			}
		}
		if _, ok := m.Files[resolver]; !ok {
			t.Errorf("%q: %s was removed", args, resolver)
		}
	}
}
