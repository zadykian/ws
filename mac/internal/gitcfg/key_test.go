package gitcfg_test

import (
	"testing"

	"github.com/zadykian/ws/mac/internal/gitcfg"
)

func TestPublicKeyIsTheOneLineOfAKey(t *testing.T) {
	for name, tc := range map[string]struct{ in, want string }{
		"ed25519":      {"ssh-ed25519 AAAAC3Nz me@mac\n", "ssh-ed25519 AAAAC3Nz"},
		"blank lines":  {"\n  ecdsa-sha2-nistp256 AAAA+/=\t c\n\n", "ecdsa-sha2-nistp256 AAAA+/="},
		"security key": {"sk-ssh-ed25519@openssh.com AAAA\n", "sk-ssh-ed25519@openssh.com AAAA"},
		"two keys":     {"ssh-ed25519 AAAA\nssh-ed25519 BBBB\n", ""},
		"private key":  {"-----BEGIN OPENSSH PRIVATE KEY-----\nAAAA\n-----END OPENSSH PRIVATE KEY-----\n", ""},
		"no data":      {"ssh-ed25519\n", ""},
		"bad data":     {"ssh-ed25519 AA_A\n", ""},
		"other type":   {"rsa AAAA\n", ""},
		"empty":        {"", ""},
	} {
		if got := gitcfg.PublicKey(tc.in); got != tc.want {
			t.Errorf("%s: PublicKey = %q, want %q", name, got, tc.want)
		}
	}
}

func TestWhyNamesWhatGhLacks(t *testing.T) {
	for name, tc := range map[string]struct {
		status int
		stderr string
		want   string
	}{
		"no login":  {4, "To get started\n", "gh has no login: gh auth login -h github.com -s write:ssh_signing_key logs in"},
		"no scope":  {1, "gh: Not Found (HTTP 404)\n", "gh's login lacks the write:ssh_signing_key scope: gh auth refresh -h github.com -s write:ssh_signing_key adds it"},
		"other":     {1, "one\nHTTP 500: Server Error.\n", "gh failed: HTTP 500: Server Error"},
		"no errors": {2, "", "gh failed: exit 2"},
	} {
		if got := gitcfg.Why(tc.status, tc.stderr); got != tc.want {
			t.Errorf("%s: Why = %q, want %q", name, got, tc.want)
		}
	}
}
