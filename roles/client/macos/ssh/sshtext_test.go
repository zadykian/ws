package ssh_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/internal/keys"
	"github.com/zadykian/ws/roles/client/macos/ssh"
	"github.com/zadykian/ws/roles/client/values"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// textMachine answers the checks of Text for the host ws, with ssh happy.
func textMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/tmp/x\n")}
	m.Results["rm -rf /tmp/x"] = effecttest.Result{}
	for _, file := range []string{"new", "whole"} {
		m.Results["ssh -T -G -F /tmp/x/"+file+" ws"] = effecttest.Result{}
	}
	return m
}

// host is ws's block at slot.
func host(slot string) sshcfg.Host {
	return sshcfg.Host{Name: "ws", Addr: values.Conf{Slot: slot}.TunnelSlot().Server(), User: "me"}
}

// TestTextNamesTheTunnelsAddress pins the literals at slots 0 and 1. The tunnel's address is part of
// the wire, so a change to the slot's end must change these lines on purpose.
func TestTextNamesTheTunnelsAddress(t *testing.T) {
	for slot, want := range map[string]string{"0": "HostName 10.99.0.1\n", "1": "HostName 10.99.1.1\n"} {
		text, err := ssh.Text(t.Context(), textMachine(), "/h", keys.NamedKey{Pub: "~/.ssh/id.pub"}, []sshcfg.Host{host(slot)})
		if err != nil || !strings.Contains(text, want) {
			t.Errorf("slot %s: Text = %q, %v; want %q", slot, text, err, want)
		}
	}
}

func TestTextRefusesAHostThatSSHReadsElsewhere(t *testing.T) {
	m := textMachine()
	m.Results["ssh -T -G -F /tmp/x/whole ws"] = effecttest.Result{Out: []byte("user me\nhostname 10.99.7.1\n")}
	_, err := ssh.Text(t.Context(), m, "/h", keys.NamedKey{Pub: "~/.ssh/id.pub"}, []sshcfg.Host{host("0")})
	if err == nil || !strings.Contains(err.Error(), "ssh reads ws as hostname 10.99.7.1") {
		t.Errorf("Text = %v, want a refusal of ws's address", err)
	}
	if _, ok := m.Files["/tmp/x/user"]; ok {
		t.Error("Text wrote the user's config with none there")
	}
}

func TestTextChecksTheUsersConfigWithoutItsInclude(t *testing.T) {
	m := textMachine()
	m.Files["/h/.ssh/config"] = []byte("Include /etc/ws/ssh_config\nHost *\n    User me\n")
	if _, err := ssh.Text(t.Context(), m, "/h", keys.NamedKey{Pub: "~/.ssh/id.pub"}, []sshcfg.Host{host("0")}); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/tmp/x/user"]); got != "\nHost *\n    User me\n" {
		t.Errorf("the user's config as ssh checked it = %q, want it without the Include", got)
	}
}
