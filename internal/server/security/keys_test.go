package security

import (
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestKeyCheckRefusesNoKeys(t *testing.T) {
	m := effecttest.NewMachine()
	k := keyCheck{m, "/home/ws"}
	if _, err := k.Plan(t.Context()); err == nil {
		t.Fatal("a missing authorized_keys passed")
	}
	m.Files["/home/ws/.ssh/authorized_keys"] = nil
	if _, err := k.Plan(t.Context()); err == nil {
		t.Error("an empty authorized_keys passed")
	}
	m.Files["/home/ws/.ssh/authorized_keys"] = []byte("ssh-ed25519 AAAA me\n")
	if _, err := k.Plan(t.Context()); err != nil {
		t.Errorf("a key was refused: %v", err)
	}
}

func TestKeyCheckPassesInAContainer(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/.dockerenv"] = nil
	if _, err := (keyCheck{m, "/home/ws"}).Plan(t.Context()); err != nil {
		t.Errorf("a container was refused: %v", err)
	}
}
