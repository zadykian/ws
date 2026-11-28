package cli

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestSSHListShowsEachKeyInATabSeparatedLine(t *testing.T) {
	p, m := sshProcess()
	one := sshPair(t, m, "ws-auth")
	two := sshPair(t, m, "ws-sign")
	m.Files[sshHome+"authorized_keys"] = []byte("ssh-ed25519 AAAA other\n")
	status, stdout, stderr := runWs(p, "", "ubuntu", "ssh", "list")
	want := "ws-auth\tssh-ed25519\t" + fingerprintOf(t, one) + "\tfile\n" +
		"ws-sign\tssh-ed25519\t" + fingerprintOf(t, two) + "\tfile\n"
	if status != 0 || stdout != want || stderr != "" {
		t.Errorf("status = %d, stdout %q, stderr %q; want stdout %q", status, stdout, stderr, want)
	}
}

func TestSSHListOfNoKeysIsEmpty(t *testing.T) {
	p, m := sshProcess()
	delete(m.Dirs, "/root/.ssh")
	for range 2 {
		if status, stdout, stderr := runWs(p, "", "ubuntu", "ssh", "list"); status != 0 || stdout != "" || stderr != "" {
			t.Errorf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
		}
		m.Dirs["/root/.ssh"] = 0o700
	}
}

func TestSSHListSkipsAPubWhoseNameBreaksTheRule(t *testing.T) {
	p, m := sshProcess()
	m.Files[sshHome+"seward.sock.pub"] = []byte("x\n")
	m.Files[sshHome+"a b.pub"] = []byte("x\n")
	m.Files[sshHome+".pub"] = []byte("x\n")
	if status, stdout, stderr := runWs(p, "", "ubuntu", "ssh", "list"); status != 0 || stdout != "" || stderr != "" {
		t.Errorf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
}

func TestSSHListMarksAPubWithNoKeyAsErrorAndExitsOne(t *testing.T) {
	p, m := sshProcess()
	m.Files[sshHome+"a.pub"] = []byte("ssh-ed25519 AAAA a\n")
	one := sshPair(t, m, "b")
	status, stdout, stderr := runWs(p, "", "ubuntu", "ssh", "list")
	want := "a\t-\t-\terror\nb\tssh-ed25519\t" + fingerprintOf(t, one) + "\tfile\n"
	if status != 1 || stdout != want || stderr != "ws ubuntu ssh list: a: no key a beside the .pub\n" {
		t.Errorf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
}

func TestSSHListMarksAnUnreadableKeyAsError(t *testing.T) {
	for name, set := range map[string]func(*testing.T, *effecttest.Machine){
		"a pub that holds no key":    func(_ *testing.T, m *effecttest.Machine) { m.Files[sshHome+"k.pub"] = []byte("junk\n") },
		"a key of no OpenSSH format": func(_ *testing.T, m *effecttest.Machine) { m.Files[sshHome+"k"] = []byte("junk\n") },
		"a link at the key":          func(_ *testing.T, m *effecttest.Machine) { m.Links[sshHome+"k"] = "/etc/ssh/ssh_host_ed25519_key" },
		"a directory in place of a pub": func(_ *testing.T, m *effecttest.Machine) {
			delete(m.Files, sshHome+"k.pub")
			m.Dirs[sshHome+"k.pub"] = 0o700
		},
	} {
		p, m := sshProcess()
		sshPair(t, m, "k")
		set(t, m)
		status, stdout, stderr := runWs(p, "", "ubuntu", "ssh", "list")
		if status != 1 || stdout != "k\t-\t-\terror\n" || !strings.HasPrefix(stderr, "ws ubuntu ssh list: k: ") || strings.Contains(stderr, "PRIVATE") {
			t.Errorf("%s: status = %d, stdout %q, stderr %q", name, status, stdout, stderr)
		}
	}
}

func TestSSHListRefusesALinkAtTheDirectory(t *testing.T) {
	p, m := sshProcess()
	m.Links["/root/.ssh"] = "/etc/ssh"
	status, stdout, stderr := runWs(p, "", "ubuntu", "ssh", "list")
	if status != 1 || stdout != "" || !strings.Contains(stderr, "is a link") {
		t.Errorf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
}
