package cli

import (
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const sshHome = "/root/.ssh/"

// sshProcess returns root's fake machine with a ~/.ssh that holds no key, and its Env, with no
// terminal.
func sshProcess() (enginecli.Env, *effecttest.Machine) {
	p := fakeProcess()
	m := effecttest.NewMachine()
	p.Machine, p.Euid = m, 0
	m.Dirs["/root"] = 0o700
	m.Dirs["/root/.ssh"] = 0o700
	return p, m
}

// sshPair puts the key NAME and its .pub in ~/.ssh, and returns the .pub's line.
func sshPair(t *testing.T, m *effecttest.Machine, name string) string {
	private, line, err := newKey(name)
	if err != nil {
		t.Fatal(err)
	}
	m.Files[sshHome+name], m.Modes[sshHome+name] = private, 0o600
	m.Files[sshHome+name+".pub"] = line
	return string(line)
}

// touched lists the calls of the machine that change it, so that a test of a refusal shows none.
func touched(m *effecttest.Machine) []string {
	var calls []string
	for _, c := range m.Calls {
		if !strings.HasPrefix(c, "stat ") && !strings.HasPrefix(c, "read home ") && !strings.HasPrefix(c, "readdir ") {
			calls = append(calls, c)
		}
	}
	return calls
}

func TestSSHRefusesABadNameAsAWrongCallBeforeAnyPath(t *testing.T) {
	// "--" ends the flags, so that a name such as -a reaches the rule.
	for _, name := range []string{"../../etc/ssh/ssh_host_ed25519_key", "a/b", "-a", "a.pub", "seward.sock", "x.lock", " "} {
		for _, verb := range []string{"keygen", "remove"} {
			p, m := sshProcess()
			status, _, stderr := runWs(p, "", "ubuntu", "ssh", verb, "--", name)
			if status != 2 || !strings.Contains(stderr, "key's name") || len(m.Calls) != 0 {
				t.Errorf("%s %q: status = %d, stderr %q, calls %v", verb, name, status, stderr, m.Calls)
			}
		}
	}
	p, m := sshProcess()
	if status, _, _ := runWs(p, "", "ubuntu", "ssh", "keygen"); status != 2 || len(m.Calls) != 0 {
		t.Errorf("keygen with no name: status = %d, calls %v", status, m.Calls)
	}
}

// fingerprintOf returns the SHA-256 fingerprint of a .pub's line.
func fingerprintOf(t *testing.T, line string) string {
	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
	if err != nil {
		t.Fatal(err)
	}
	return ssh.FingerprintSHA256(pub)
}
