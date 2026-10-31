package sshcfg_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/sshcfg"
)

func inputs() sshcfg.Inputs {
	return sshcfg.Inputs{
		User: "me", LAN: "192.168.1.0/24", LANAddress: "192.168.1.5", Remote: "ws.example.org",
		RemotePort: "2222", Patterns: "UseKeychain", IdentityPub: "~/.ssh/id_ed25519.pub",
	}
}

func TestConfigBeforeTunnel(t *testing.T) {
	got := sshcfg.Config(inputs())
	for _, want := range []string{
		"IgnoreUnknown UseKeychain\n",
		"Match originalhost ws,ws-ssh localnetwork 10.77.0.0/30\n    HostName 10.77.0.1\n    Port 22\n",
		"Match originalhost ws,ws-ssh localnetwork 192.168.1.0/24\n    HostName 192.168.1.5\n",
		"Host ws ws-ssh\n    HostName ws.example.org\n    Port 2222\n    AddressFamily inet\n",
		"    IdentityFile ~/.ssh/id_ed25519\n    IdentitiesOnly yes\n    UseKeychain yes\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Config lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "HostName 10.99.0.1") {
		t.Errorf("Config holds the tunnel or cld's lines:\n%s", got)
	}
}

func TestConfigViaTunnelWithAgent(t *testing.T) {
	in := inputs()
	in.ViaTunnel, in.LAN, in.RemotePort = true, "", ""
	in.IdentityPub, in.IdentityAgent = "~/k.pub", "~/agent.sock"
	got := sshcfg.Config(in)
	for _, want := range []string{
		"Match originalhost ws-ssh localnetwork 10.77.0.0/30\n",
		"Host ws\n    HostName 10.99.0.1\n    Port 22\n",
		"    IdentityFile ~/k.pub\n    IdentitiesOnly yes\n    IdentityAgent \"~/agent.sock\"\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Config lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "192.168") || strings.Contains(got, "AddressFamily") {
		t.Errorf("Config holds a LAN or a forward that mac.conf lacks:\n%s", got)
	}
}

func TestConfigCldTabs(t *testing.T) {
	in := inputs()
	in.CldTabs = true
	got := sshcfg.Config(in)
	if !strings.Contains(got, "    PermitLocalCommand yes\n    LocalCommand [ \"$TERM_PROGRAM\" != ghostty ]") {
		t.Errorf("Config lacks cld's lines:\n%s", got)
	}
}
