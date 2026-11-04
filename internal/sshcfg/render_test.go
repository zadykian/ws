package sshcfg_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/sshcfg"
)

func inputs() sshcfg.Inputs {
	return sshcfg.Inputs{User: "me", Patterns: "UseKeychain", IdentityPub: "~/.ssh/id_ed25519.pub"}
}

func TestConfigHoldsOneHostAtTheTunnelsEnd(t *testing.T) {
	got := sshcfg.Config(inputs())
	for _, want := range []string{
		"IgnoreUnknown UseKeychain\n",
		"Host ws\n    HostName 10.99.0.1\n    Port 22\n    User me\n    HostKeyAlias ws\n",
		"    IdentityFile ~/.ssh/id_ed25519\n    IdentitiesOnly yes\n    UseKeychain yes\n",
		"    ServerAliveInterval 30\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Config lacks %q:\n%s", want, got)
		}
	}
	if n := strings.Count(got, "\nHost "); n != 1 || strings.Contains(got, "Match") || strings.Contains(got, "PermitLocalCommand") {
		t.Errorf("Config holds %d hosts, a Match or cld's lines; want one host alone:\n%s", n, got)
	}
}

func TestConfigWithAgent(t *testing.T) {
	in := inputs()
	in.IdentityPub, in.IdentityAgent = "~/k.pub", "~/agent.sock"
	got := sshcfg.Config(in)
	want := "    IdentityFile ~/k.pub\n    IdentitiesOnly yes\n    IdentityAgent \"~/agent.sock\"\n"
	if !strings.Contains(got, want) {
		t.Errorf("Config lacks %q:\n%s", want, got)
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
