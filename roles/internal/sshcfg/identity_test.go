package sshcfg_test

import (
	"testing"

	"github.com/zadykian/ws/roles/internal/sshcfg"
)

const home = "/Users/me"

func TestNamedIdentity(t *testing.T) {
	data := "~/" + sshcfg.SecretiveData
	tests := []struct {
		name, pub, agent string
		want             sshcfg.Identity
	}{
		{"a file key", "~/k.pub", "", sshcfg.Identity{Pub: "~/k.pub"}},
		{"a file key and its agent", "~/k.pub", "~/a.sock", sshcfg.Identity{Pub: "~/k.pub", Agent: "~/a.sock"}},
		{"ws's copy", "~/" + sshcfg.KeyCopies + "/b.pub", "",
			sshcfg.Identity{Pub: "~/" + sshcfg.KeyCopies + "/b.pub", Agent: data + "/socket.ssh"}},
		{"a .pub in the container", data + "/PublicKeys/b.pub", "",
			sshcfg.Identity{Pub: data + "/PublicKeys/b.pub", Agent: data + "/socket.ssh"}},
		{"another agent", data + "/PublicKeys/b.pub", "~/a.sock",
			sshcfg.Identity{Pub: data + "/PublicKeys/b.pub", Agent: "~/a.sock"}},
	}
	for _, tt := range tests {
		if got := sshcfg.NamedIdentity(tt.pub, tt.agent, home); got != tt.want {
			t.Errorf("%s: NamedIdentity = %+v; want %+v", tt.name, got, tt.want)
		}
	}
}

// The name is what Secretive gives the key's .pub, as ssh-keygen -E md5 -l prints its fingerprint.
func TestAgentKeys(t *testing.T) {
	const key = "ecdsa-sha2-nistp256 AAAA me"
	list := key + "\necdsa-sha2-nistp256-cert-v01@openssh.com AAAA me\nThe agent has no identities.\n\nssh-rsa ! me\n"
	got := sshcfg.AgentKeys(list)
	if len(got) != 1 || got[0].Line != key || got[0].Name != "693e9af84d3dfcc71e640e005bdc5e2e.pub" {
		t.Errorf("AgentKeys = %+v", got)
	}
	if want := "~/.config/ws/secretive/" + got[0].Name; got[0].Pub() != want {
		t.Errorf("Pub = %q, want %q", got[0].Pub(), want)
	}
}

func TestSecretiveKey(t *testing.T) {
	for pub, want := range map[string]bool{
		home + "/" + sshcfg.SecretiveData + "/PublicKeys/a.pub": true,
		home + "/" + sshcfg.KeyCopies + "/a.pub":                true,
		home + "/" + sshcfg.SecretiveData + "/a.pub":            false,
		home + "/.ssh/a.pub":                                    false,
	} {
		if got := sshcfg.SecretiveKey(pub, home); got != want {
			t.Errorf("SecretiveKey(%q) = %v", pub, got)
		}
	}
}
