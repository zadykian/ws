package cli

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestRemoveDeletesTheServersFiles(t *testing.T) {
	p := withEnv(fakeProcess(), map[string]string{"WS_CONF": "/e/mac.conf", "TMPDIR": "/t"})
	m := p.Machine.(*effecttest.Machine)
	m.Files["/e/servers/ws/server.conf"] = []byte("WS_USER=ada\n")
	m.Files["/e/servers/lab/server.conf"] = []byte("WS_USER=ada\n")
	withServerDirs(m)
	m.Results["rm -rf /e/servers/lab"] = effecttest.Result{}
	m.Results["rm -rf /t/ws-wizard.lock"] = effecttest.Result{}
	status, stdout, stderr := runWs(p, "", "client", "macos", "remove", "lab")
	if status != 0 || stdout != "removed lab\n" || stderr != "" {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want 0, removed lab", status, stdout, stderr)
	}
	if !slices.Contains(m.Calls, "run rm -rf /e/servers/lab") || slices.Contains(m.Calls, "run rm -rf /e/servers/ws") {
		t.Errorf("calls %q: want lab's files removed and ws's kept", m.Calls)
	}
}
