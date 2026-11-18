package cli

import (
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
	"github.com/zadykian/ws/roles/client/macos/values"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// cldLatest is the read of cld's latest release.
const cldLatest = "curl -fsSIL --retry 3 -o /dev/null -w %{url_effective} https://github.com/zadykian/cld/releases/latest"

// partEnv returns a run of ws client macos as the user of /h, on the machine m.
func partEnv(m *effecttest.Machine) func(args ...string) (int, string, string) {
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = m
	return func(args ...string) (int, string, string) {
		return runWs(p, append([]string{"client", "macos"}, args...)...)
	}
}

func TestCldCheckExitsOneWhereCldIsMissing(t *testing.T) {
	m := effecttest.NewMachine()
	status, stdout, stderr := partEnv(m)("cld", "--check")
	if status != 1 || stdout != "" || !strings.Contains(stderr, "would install cld") {
		t.Errorf("cld --check: status = %d, stdout = %q, stderr = %q; want 1, the install due", status, stdout, stderr)
	}
	for _, c := range m.Calls {
		if strings.Contains(c, "install.sh") {
			t.Errorf("--check ran %q, want nothing run", c)
		}
	}
}

func TestCldCheckFollowsTheLatestRelease(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/.local/bin/cld"], m.Modes["/h/.local/bin/cld"] = nil, 0o755
	m.Results["/h/.local/bin/cld --version"] = effecttest.Result{Out: []byte("cld 0.13.0\n")}
	m.Results["/h/.local/bin/cld help update"] = effecttest.Result{}
	m.Results[cldLatest] = effecttest.Result{Out: []byte("https://github.com/zadykian/cld/releases/tag/v0.13.0")}
	run := partEnv(m)
	if status, stdout, stderr := run("cld", "--check"); status != 0 || stdout != "" {
		t.Errorf("at the latest: status = %d, stdout = %q, stderr = %q; want 0", status, stdout, stderr)
	}
	m.Results[cldLatest] = effecttest.Result{Out: []byte("https://github.com/zadykian/cld/releases/tag/v0.14.0")}
	if status, _, stderr := run("cld", "--check"); status != 1 || !strings.Contains(stderr, "the latest 0.14.0") {
		t.Errorf("behind: status = %d, stderr = %q; want 1, the update due", status, stderr)
	}
	m.Results["sh -c exec \"$0\" update 2>&1 /h/.local/bin/cld"] = effecttest.Result{Out: []byte("Updated to 0.14.0\n")}
	if status, _, stderr := run("cld"); status != 0 || !slices.Contains(m.Calls, "run sh -c exec \"$0\" update 2>&1 /h/.local/bin/cld") {
		t.Errorf("run: status = %d, stderr = %q, calls %q; want cld update run", status, stderr, m.Calls)
	}
}

func TestKeysCopyTheAgentsKeyOnce(t *testing.T) {
	m := effecttest.NewMachine()
	socket := "/h/" + sshcfg.SecretiveData + "/socket.ssh"
	m.Files[socket], m.Modes[socket] = nil, fs.ModeSocket
	m.Results["env SSH_AUTH_SOCK="+socket+" ssh-add -L"] = effecttest.Result{Out: []byte(mactest.AgentGit)}
	pub := sshcfg.AgentKeys(mactest.AgentGit)[0].Pub()
	m.Files["/etc/ws/values.yml"] = []byte(values.Text("WS_IDENTITY_PUB_KEY", pub, "WS_GIT_SIGNING_PUB_KEY", pub))
	run := partEnv(m)
	if status, stdout, stderr := run("keys", "--check"); status != 1 || stdout != "" || strings.Count(stderr, pub) != 1 {
		t.Errorf("keys --check: status = %d, stdout = %q, stderr = %q; want 1, one copy due", status, stdout, stderr)
	}
	if status, _, stderr := run("keys"); status != 0 || string(m.Files[pub]) != mactest.AgentGit {
		t.Errorf("keys: status = %d, stderr = %q, copy %q; want the agent's line", status, stderr, m.Files[pub])
	}
	if status, _, stderr := run("keys", "--check"); status != 0 {
		t.Errorf("second keys --check: status = %d, stderr = %q; want 0", status, stderr)
	}
}

func TestKeysOfAFileKeyHaveNoCopy(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/etc/ws/values.yml"] = []byte(values.Text("WS_IDENTITY_PUB_KEY", "~/.ssh/id.pub", "WS_GIT_SIGNING_PUB_KEY", "~/.ssh/id.pub"))
	m.Files["/h/.ssh/id.pub"] = []byte("ssh-ed25519 AAAA me\n")
	m.Results["ssh-keygen -l -f /h/.ssh/id.pub"] = effecttest.Result{}
	if status, stdout, stderr := partEnv(m)("keys", "--check"); status != 0 || stdout != "" {
		t.Errorf("keys --check: status = %d, stdout = %q, stderr = %q; want 0", status, stdout, stderr)
	}
}
