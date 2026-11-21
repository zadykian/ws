package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/values"
)

// badSiblingFixture is ws beside lab, whose file is refused for a slot out of range, and the
// resolver file that lab's run made.
func badSiblingFixture() (*effecttest.Machine, string) {
	m := effecttest.NewMachine()
	keyed(m)
	m.Files["/etc/ws/servers/lab/values.yml"] = []byte(values.Text("WS_USER", "ada", "WS_SLOT", "99"))
	m.Dirs["/etc/ws/servers/lab"] = 0o755
	m.Dirs["/etc/resolver"] = 0o755
	resolver := "/etc/resolver/lab.test"
	m.Files[resolver] = []byte("# ws\nnameserver 10.30.0.53\n")
	return m, resolver
}

// TestRefuseBadSiblingReadsAlone settles R1071-3 for remove: a step could prune what a server it
// cannot read holds. So the guard refuses lab's file for ws, names remove, and writes nothing.
func TestRefuseBadSiblingReadsAlone(t *testing.T) {
	m, resolver := badSiblingFixture()
	err := refuseBadSibling(m, "/etc/ws/values.yml", "ws")
	if err == nil || !strings.Contains(err.Error(), "server lab: ") || !strings.Contains(err.Error(), "fix its file, or ws client remove lab") {
		t.Errorf("err = %v; want lab's file refused, naming remove", err)
	}
	for _, call := range m.Calls {
		if !slices.Contains([]string{"read", "readdir", "stat"}, strings.Fields(call)[0]) {
			t.Errorf("call %q ran", call)
		}
	}
	if _, ok := m.Files[resolver]; !ok {
		t.Errorf("%s was removed", resolver)
	}
	if err := refuseBadSibling(m, "/etc/ws/values.yml", "lab"); err != nil {
		t.Errorf("lab's own file: %v", err)
	}
}
