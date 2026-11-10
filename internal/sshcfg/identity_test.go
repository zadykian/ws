package sshcfg_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/sshcfg"
)

const home = "/Users/me"

func TestPickIdentity(t *testing.T) {
	data := "~/" + sshcfg.SecretiveData
	tests := []struct {
		name, identity, agent string
		socket                bool
		keys                  []string
		want                  sshcfg.Identity
	}{
		{"the file key", "", "", false, nil, sshcfg.Identity{Pub: sshcfg.DefaultPub}},
		{"no socket, so the file key", "", "", false, []string{"a.pub"}, sshcfg.Identity{Pub: sshcfg.DefaultPub}},
		{"mac.conf's key", "~/k.pub", "", true, []string{"a.pub"}, sshcfg.Identity{Pub: "~/k.pub", Check: true}},
		{"Secretive's one key", "", "", true, []string{"a.pub", "sign.pub"},
			sshcfg.Identity{Pub: "~/" + sshcfg.KeyCopies + "/a.pub", Agent: data + "/socket.ssh", Check: true}},
		{"a copy of mac.conf's", "~/" + sshcfg.KeyCopies + "/b.pub", "", true, nil,
			sshcfg.Identity{Pub: "~/" + sshcfg.KeyCopies + "/b.pub", Agent: data + "/socket.ssh", Check: true}},
		{"a .pub in the container", data + "/PublicKeys/b.pub", "", true, nil,
			sshcfg.Identity{Pub: data + "/PublicKeys/b.pub", Agent: data + "/socket.ssh", Check: true}},
	}
	for _, tt := range tests {
		got, err := sshcfg.PickIdentity(tt.identity, tt.agent, "~/Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data/PublicKeys/sign.pub",
			home, tt.socket, tt.keys)
		if err != nil || got != tt.want {
			t.Errorf("%s: PickIdentity = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
	_, err := sshcfg.PickIdentity("", "", "", home, true, []string{"a.pub", "b.pub"})
	if err == nil || !strings.Contains(err.Error(), "several keys") {
		t.Errorf("two keys: err = %v", err)
	}
}

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
		tt.want.Check = true
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
