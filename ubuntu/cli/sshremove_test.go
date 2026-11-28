package cli

import (
	"strings"
	"testing"
)

// held lists the files of ~/.ssh that the machine keeps.
func held(files map[string][]byte) []string {
	var list []string
	for f := range files {
		if strings.HasPrefix(f, sshHome) {
			list = append(list, strings.TrimPrefix(f, sshHome))
		}
	}
	return list
}

func TestSSHRemoveWithYesDeletesBothFilesAndNothingElse(t *testing.T) {
	p, m := sshProcess()
	sshPair(t, m, "k")
	sshPair(t, m, "other")
	status, stdout, stderr := runWs(p, "", "ubuntu", "ssh", "remove", "--yes", "k")
	if status != 0 || stdout != "" || !strings.Contains(stderr, "remove /root/.ssh/k") {
		t.Errorf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
	if got := strings.Join(held(m.Files), " "); len(m.Files) != 2 || !strings.Contains(got, "other") || strings.Contains(got, "k.pub") {
		t.Errorf("files left: %v", held(m.Files))
	}
}

func TestSSHRemoveDeletesAPubAlone(t *testing.T) {
	p, m := sshProcess()
	m.Files[sshHome+"k.pub"] = []byte("ssh-ed25519 AAAA k\n")
	if status, _, stderr := runWs(p, "", "ubuntu", "ssh", "remove", "--yes", "k"); status != 0 || len(m.Files) != 0 {
		t.Errorf("status = %d, stderr %q, files %v", status, stderr, held(m.Files))
	}
}

func TestSSHRemoveWhereNeitherFileExistsExitsOne(t *testing.T) {
	p, m := sshProcess()
	status, _, stderr := runWs(p, "", "ubuntu", "ssh", "remove", "--yes", "k")
	if status != 1 || !strings.Contains(stderr, "no key k") || len(touched(m)) != 0 {
		t.Errorf("status = %d, stderr %q, touched %v", status, stderr, touched(m))
	}
}

func TestSSHRemoveWithNoTerminalAndNoYesIsAWrongCall(t *testing.T) {
	p, m := sshProcess()
	sshPair(t, m, "k")
	status, _, stderr := runWs(p, "k\n", "ubuntu", "ssh", "remove", "k")
	if status != 2 || !strings.Contains(stderr, "pass --yes") || len(m.Files) != 2 || len(m.Calls) != 0 {
		t.Errorf("status = %d, stderr %q, files %v, calls %v", status, stderr, held(m.Files), m.Calls)
	}
}

func TestSSHRemoveAsksForTheNameAgain(t *testing.T) {
	for _, c := range []struct {
		answer  string
		deleted bool
	}{{"k\n", true}, {"k", true}, {" k \n", true}, {"\n", false}, {"yes\n", false}, {"K\n", false}, {"", false}} {
		answer, deleted := c.answer, c.deleted
		p, m := sshProcess()
		p.Terminal = true
		sshPair(t, m, "k")
		status, _, stderr := runWs(p, answer, "ubuntu", "ssh", "remove", "k")
		if !strings.Contains(stderr, "Type k to confirm") {
			t.Errorf("answer %q: no question: %q", answer, stderr)
		}
		if gone := len(m.Files) == 0; gone != deleted || (status == 0) != deleted {
			t.Errorf("answer %q: status = %d, files %v; want deleted %v", answer, status, held(m.Files), deleted)
		}
	}
}

func TestSSHRemoveDeletesNothingWhereTheNameIsNoPrivateKey(t *testing.T) {
	p, m := sshProcess()
	m.Files[sshHome+"k"] = []byte("my notes\n")
	m.Files[sshHome+"k.pub"] = []byte("ssh-ed25519 AAAA k\n")
	status, _, stderr := runWs(p, "", "ubuntu", "ssh", "remove", "--yes", "k")
	if status != 1 || !strings.Contains(stderr, "no OpenSSH private key") || len(m.Files) != 2 || len(touched(m)) != 0 {
		t.Errorf("status = %d, stderr %q, files %v, touched %v", status, stderr, held(m.Files), touched(m))
	}
}

func TestSSHRemoveRefusesALink(t *testing.T) {
	for _, link := range []string{"/root/.ssh", "/root/.ssh/k", "/root/.ssh/k.pub"} {
		p, m := sshProcess()
		sshPair(t, m, "k")
		m.Links[link] = "/etc/ssh"
		status, _, stderr := runWs(p, "", "ubuntu", "ssh", "remove", "--yes", "k")
		if status != 1 || !strings.Contains(stderr, "is a link") || len(m.Files) != 2 || len(touched(m)) != 0 {
			t.Errorf("link at %s: status = %d, stderr %q, files %v, touched %v", link, status, stderr, held(m.Files), touched(m))
		}
	}
}
