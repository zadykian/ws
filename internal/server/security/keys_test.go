package security

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func TestKeyCheckRefusesNoKeys(t *testing.T) {
	m := effecttest.NewMachine()
	m.Root["/home/ws"] = effect.Attrs{UID: 1000, GID: 1000, Mode: 0o755}
	k := keyCheck{m, host.User{Name: "ws", Home: "/home/ws"}}
	if _, err := k.Plan(t.Context()); err == nil {
		t.Fatal("a missing authorized_keys passed")
	}
	m.Files["/home/ws/.ssh/authorized_keys"] = nil
	if _, err := k.Plan(t.Context()); err == nil {
		t.Error("an empty authorized_keys passed")
	}
	m.Files["/home/ws/.ssh/authorized_keys"] = []byte("# none yet\n\n")
	if _, err := k.Plan(t.Context()); err == nil {
		t.Error("an authorized_keys of comments passed")
	}
}

// sshd takes the user's file, or root's where the user can read it, with no other able to write.
func TestKeyCheckRefusesAFileSSHDPassesOver(t *testing.T) {
	const file = "/home/ws/.ssh/authorized_keys"
	for _, c := range []struct {
		name string
		a    effect.Attrs
		ok   bool
	}{
		{"the user's", effect.Attrs{UID: 1000, Mode: 0o600}, true},
		{"root's, open to read", effect.Attrs{Mode: 0o644}, true},
		{"root's, as cp leaves it", effect.Attrs{Mode: 0o600}, false},
		{"another user's", effect.Attrs{UID: 1001, Mode: 0o644}, false},
		{"open to the group to write", effect.Attrs{UID: 1000, Mode: 0o660}, false},
	} {
		m := effecttest.NewMachine()
		m.Root["/home/ws"] = effect.Attrs{UID: 1000, GID: 1000, Mode: 0o755}
		m.Root[file] = c.a
		m.Files[file] = []byte("ssh-ed25519 AAAA me\n")
		_, err := keyCheck{m, host.User{Name: "ws", Home: "/home/ws"}}.Plan(t.Context())
		if (err == nil) != c.ok {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

func TestKeyCheckPassesInAContainer(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/.dockerenv"] = nil
	if _, err := (keyCheck{m, host.User{Name: "ws", Home: "/home/ws"}}).Plan(t.Context()); err != nil {
		t.Errorf("a container was refused: %v", err)
	}
}

// The user owns the home, so a link at authorized_keys or above it would send root's read elsewhere.
func TestKeyCheckRefusesALink(t *testing.T) {
	for _, link := range []string{"/home/ws/.ssh/authorized_keys", "/home/ws/.ssh"} {
		m := effecttest.NewMachine()
		m.Files["/home/ws/.ssh/authorized_keys"] = []byte("ssh-ed25519 AAAA me\n")
		m.Root["/home/ws/.ssh/authorized_keys"] = effect.Attrs{UID: 1000, Mode: 0o600}
		m.Links[link] = "/srv/target"
		if _, err := (keyCheck{m, host.User{Name: "ws", Home: "/home/ws"}}).Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "link") {
			t.Errorf("a link at %s: %v", link, err)
		}
	}
}
