package cli

import (
	"slices"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// removeFixture is ws and lab on a Mac, whose known_hosts holds both, and the commands that a removal runs.
func removeFixture() (enginecli.Env, *effecttest.Machine) {
	p := withEnv(fakeProcess(), map[string]string{"WS_CONF": "/e/mac.conf", "HOME": "/h", "TMPDIR": "/t"})
	m := p.Machine.(*effecttest.Machine)
	m.Files["/e/servers/ws/server.conf"] = []byte("WS_USER=ada\n")
	m.Files["/e/servers/lab/server.conf"] = []byte("WS_USER=ada\n")
	withServerDirs(m)
	m.Files["/h/.ssh/known_hosts"] = []byte("ws ssh-ed25519 AAAA\nlab ssh-ed25519 BBBB\n")
	for _, line := range []string{"rm -rf /e/servers/lab", "rm -rf /t/ws-wizard.lock", "ssh-keygen -R lab"} {
		m.Results[line] = effecttest.Result{}
	}
	return p, m
}

func TestRemoveDeletesTheServersFilesAndItsHostKey(t *testing.T) {
	p, m := removeFixture()
	status, stdout, stderr := runWs(p, "", "client", "macos", "remove", "lab")
	if status != 0 || stdout != "removed lab\n" || stderr != "" {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want 0, removed lab", status, stdout, stderr)
	}
	for _, want := range []string{"run rm -rf /e/servers/lab", "run ssh-keygen -R lab"} {
		if !slices.Contains(m.Calls, want) {
			t.Errorf("calls %q lack %q", m.Calls, want)
		}
	}
	if slices.Contains(m.Calls, "run rm -rf /e/servers/ws") {
		t.Error("remove deleted ws's files too")
	}
}

func TestRemoveLeavesAKnownHostsThatIsALink(t *testing.T) {
	p, m := removeFixture()
	m.Links["/h/.ssh/known_hosts"] = "/d/known_hosts"
	m.Files["/d/known_hosts"] = []byte("lab ssh-ed25519 BBBB\n")
	_, stdout, _ := runWs(p, "", "client", "macos", "remove", "lab")
	if want := "By hand: /h/.ssh/known_hosts is a link to /d/known_hosts: delete the lines of lab there\nremoved lab\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
	if slices.Contains(m.Calls, "run ssh-keygen -R lab") {
		t.Error("remove ran ssh-keygen over a link")
	}
}

func TestRemoveWithNoKnownHostsHasNoKeyToDrop(t *testing.T) {
	p, m := removeFixture()
	delete(m.Files, "/h/.ssh/known_hosts")
	if status, _, stderr := runWs(p, "", "client", "macos", "remove", "lab"); status != 0 || stderr != "" {
		t.Errorf("status = %d, stderr = %q; want 0", status, stderr)
	}
}
