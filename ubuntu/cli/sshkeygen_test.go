package cli

import (
	"crypto/ed25519"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestSSHKeygenMakesAnEd25519KeyWithItsPub(t *testing.T) {
	p, m := sshProcess()
	delete(m.Dirs, "/root/.ssh")
	status, stdout, stderr := runWs(p, "", "ubuntu", "ssh", "keygen", "ws-sign")
	if status != 0 || stdout != "" {
		t.Fatalf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
	if m.Dirs["/root/.ssh"] != 0o700 || m.Modes[sshHome+"ws-sign"] != 0o600 || m.Modes[sshHome+"ws-sign.pub"] != 0o644 {
		t.Errorf("modes: dir %o, key %o, pub %o", m.Dirs["/root/.ssh"], m.Modes[sshHome+"ws-sign"], m.Modes[sshHome+"ws-sign.pub"])
	}
	for _, f := range []string{"ws-sign", "ws-sign.pub"} {
		if o := m.Owners[sshHome+f]; o != [2]int{0, 0} {
			t.Errorf("%s: owner %v, want root", f, o)
		}
	}
	raw, err := ssh.ParseRawPrivateKey(m.Files[sshHome+"ws-sign"])
	if err != nil {
		t.Fatalf("the key does not parse with no passphrase: %v", err)
	}
	if _, ok := raw.(*ed25519.PrivateKey); !ok {
		t.Errorf("key is %T, want an ed25519 key", raw)
	}
	signer, err := ssh.NewSignerFromKey(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSuffix(string(ssh.MarshalAuthorizedKey(signer.PublicKey())), "\n") + " ws-sign\n"
	if got := string(m.Files[sshHome+"ws-sign.pub"]); got != want {
		t.Errorf("pub = %q, want %q", got, want)
	}
	if strings.Contains(stderr, "PRIVATE KEY") || len(m.Inputs) != 0 {
		t.Errorf("stderr %q, inputs %q", stderr, m.Inputs)
	}
}

func TestSSHKeygenLeavesAnExistingDirectoryAlone(t *testing.T) {
	p, m := sshProcess()
	m.Dirs["/root/.ssh"] = 0o755
	if status, _, stderr := runWs(p, "", "ubuntu", "ssh", "keygen", "k"); status != 0 {
		t.Fatalf("status = %d, stderr %q", status, stderr)
	}
	if m.Dirs["/root/.ssh"] != 0o755 {
		t.Errorf("dir mode = %o, want it kept", m.Dirs["/root/.ssh"])
	}
}

func TestSSHKeygenWithBothFilesChangesNothing(t *testing.T) {
	p, m := sshProcess()
	pub := sshPair(t, m, "k")
	key := string(m.Files[sshHome+"k"])
	status, _, stderr := runWs(p, "", "ubuntu", "ssh", "keygen", "k")
	if status != 1 || !strings.Contains(stderr, "exist already") {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
	if string(m.Files[sshHome+"k"]) != key || string(m.Files[sshHome+"k.pub"]) != pub || len(touched(m)) != 0 {
		t.Errorf("the files or the machine changed: %v", touched(m))
	}
}

func TestSSHKeygenWithTheKeyAloneWritesThePubByKeygenY(t *testing.T) {
	p, m := sshProcess()
	pub := sshPair(t, m, "k")
	delete(m.Files, sshHome+"k.pub")
	fields := strings.Fields(pub)
	m.Results["ssh-keygen -y -P  -f /root/.ssh/k"] = effecttest.Result{Out: []byte(fields[0] + " " + fields[1] + " comment-of-the-key\n")}
	status, _, stderr := runWs(p, "", "ubuntu", "ssh", "keygen", "k")
	if status != 0 {
		t.Fatalf("status = %d, stderr %q", status, stderr)
	}
	if got := string(m.Files[sshHome+"k.pub"]); got != pub || m.Modes[sshHome+"k.pub"] != 0o644 {
		t.Errorf("pub = %q at %o, want %q", got, m.Modes[sshHome+"k.pub"], pub)
	}
}

func TestSSHKeygenWithTheKeyAloneRefusesAFileThatIsNoKey(t *testing.T) {
	p, m := sshProcess()
	m.Files[sshHome+"k"] = []byte("not a key\n")
	status, _, stderr := runWs(p, "", "ubuntu", "ssh", "keygen", "k")
	if status != 1 || !strings.Contains(stderr, "no OpenSSH private key") || len(touched(m)) != 0 {
		t.Errorf("status = %d, stderr %q, touched %v", status, stderr, touched(m))
	}
}

func TestSSHKeygenWithThePubAloneExitsOneAndSaysRemove(t *testing.T) {
	p, m := sshProcess()
	m.Files[sshHome+"k.pub"] = []byte("ssh-ed25519 AAAA k\n")
	status, _, stderr := runWs(p, "", "ubuntu", "ssh", "keygen", "k")
	if status != 1 || !strings.Contains(stderr, "run remove k") || len(touched(m)) != 0 {
		t.Errorf("status = %d, stderr %q, touched %v", status, stderr, touched(m))
	}
}

func TestSSHKeygenRefusesALink(t *testing.T) {
	for name, link := range map[string]string{"/root/.ssh": "/etc/ssh", "/root/.ssh/k": "/etc/ssh/ssh_host_ed25519_key", "/root/.ssh/k.pub": "/tmp/x"} {
		p, m := sshProcess()
		m.Links[name] = link
		status, _, stderr := runWs(p, "", "ubuntu", "ssh", "keygen", "k")
		if status != 1 || !strings.Contains(stderr, "is a link") || len(touched(m)) != 0 {
			t.Errorf("link at %s: status = %d, stderr %q, touched %v", name, status, stderr, touched(m))
		}
	}
}

// OpenSSH itself reads a new key: ssh-keygen derives the same public key and prints the
// fingerprint that list shows. The key is the test's own, under its temp dir.
func TestSSHKeygenMakesAKeyThatOpenSSHReads(t *testing.T) {
	tool, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("no ssh-keygen on this machine")
	}
	private, line, err := newKey("k")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "k")
	if err := os.WriteFile(file, private, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := exec.Command(tool, "-y", "-P", "", "-f", file).Output() //nolint:gosec // the test's own tool and its own key
	if err != nil {
		t.Fatalf("ssh-keygen -y: %v", err)
	}
	if string(got) != string(line) {
		t.Errorf("ssh-keygen -y = %q, the .pub is %q", got, line)
	}
	if err := os.WriteFile(file+".pub", line, 0o644); err != nil { //nolint:gosec // a .pub is public, 0644 as ws sets it
		t.Fatal(err)
	}
	listed, err := exec.Command(tool, "-l", "-f", file+".pub").Output() //nolint:gosec // the test's own tool and its own key
	if err != nil {
		t.Fatalf("ssh-keygen -l: %v", err)
	}
	if !strings.Contains(string(listed), fingerprintOf(t, string(line))) {
		t.Errorf("ssh-keygen -l = %q, want the fingerprint %s", listed, fingerprintOf(t, string(line)))
	}
}

// racer plants a file at the first WriteNew, as a keygen of the same name does in the same instant.
type racer struct {
	*effecttest.Machine
	planted bool
}

func (r *racer) WriteNew(path string, data []byte, a effect.Attrs) error {
	if !r.planted {
		r.planted = true
		r.Files[path] = []byte("theirs\n")
	}
	return r.Machine.WriteNew(path, data, a)
}

func TestSSHKeygenKeepsTheKeyThatAnotherRunMadeFirst(t *testing.T) {
	p, m := sshProcess()
	p.Machine = &racer{Machine: m}
	status, _, stderr := runWs(p, "", "ubuntu", "ssh", "keygen", "k")
	if status != 1 || !strings.Contains(stderr, "exist") {
		t.Errorf("status = %d, stderr %q; want 1 and the refusal", status, stderr)
	}
	if got := string(m.Files[sshHome+"k"]); got != "theirs\n" {
		t.Errorf("key = %q, want the other run's kept", got)
	}
	if _, ok := m.Files[sshHome+"k.pub"]; ok {
		t.Error("a .pub was written beside the other run's key")
	}
}
