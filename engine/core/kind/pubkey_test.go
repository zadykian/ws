package kind

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/check"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// renderPub renders ssh_pub of user and path on m.
func renderPub(t *testing.T, m *effecttest.Machine, user, path string) (string, error) {
	dir := fstest.MapFS{"t.tmpl": {Data: []byte(`{{ ssh_pub "` + user + `" "` + path + `" }}`)}}
	got, err := render(t.Context(), Env{M: m}, dir, "t.tmpl")
	return string(got), err
}

func TestSSHPubIsTheKeysTypeAndData(t *testing.T) {
	m := effecttest.NewMachine()
	withAccount(m)
	m.Dirs["/home/u/.ssh"] = 0o700
	m.Files["/home/u/.ssh/k.pub"] = []byte("ssh-ed25519 AAAAC3Nza u@ws\n")
	if got, err := renderPub(t, m, "u", "/home/u/.ssh/k.pub"); err != nil || got != "ssh-ed25519 AAAAC3Nza" {
		t.Errorf("ssh_pub = %q, %v; want the type and data, the comment dropped", got, err)
	}
}

func TestSSHPubOfAMissingFileIsEmpty(t *testing.T) {
	m := effecttest.NewMachine()
	withAccount(m)
	if got, err := renderPub(t, m, "u", "/home/u/.ssh/k.pub"); err != nil || got != "" {
		t.Errorf("ssh_pub = %q, %v; want empty", got, err)
	}
}

func TestSSHPubRefusesALinkOnTheWay(t *testing.T) {
	m := effecttest.NewMachine()
	withAccount(m)
	m.Links["/home/u/.ssh"] = "/etc/ssh"
	m.Files["/etc/ssh/k.pub"] = []byte("ssh-ed25519 AAAA\n")
	if got, err := renderPub(t, m, "u", "/home/u/.ssh/k.pub"); err == nil {
		t.Errorf("ssh_pub = %q, want a link's refusal", got)
	}
}

func TestSSHPubRefusesTwoKeysAndAPathOutsideTheHome(t *testing.T) {
	m := effecttest.NewMachine()
	withAccount(m)
	m.Files["/home/u/k.pub"] = []byte("ssh-ed25519 AAAA\nssh-ed25519 BBBB\n")
	if _, err := renderPub(t, m, "u", "/home/u/k.pub"); !errors.Is(err, check.ErrNotAKey) {
		t.Errorf("two keys: ssh_pub = %v, want ErrNotAKey", err)
	}
	if _, err := renderPub(t, m, "u", "/etc/k.pub"); err == nil || !strings.Contains(err.Error(), "is not below /home/u") {
		t.Errorf("outside the home: ssh_pub = %v", err)
	}
}

func TestSSHPubOfRootReadsRootsHome(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["getent passwd root"] = effecttest.Result{Out: []byte("root:x:0:0:root:/root:/bin/bash\n")}
	m.Dirs["/root"], m.Dirs["/root/.ssh"] = 0o700, 0o700
	m.Files["/root/.ssh/k.pub"] = []byte("ssh-ed25519 RRRR root@ws\n")
	if got, err := renderPub(t, m, "root", "/root/.ssh/k.pub"); err != nil || got != "ssh-ed25519 RRRR" {
		t.Errorf("ssh_pub = %q, %v", got, err)
	}
}
