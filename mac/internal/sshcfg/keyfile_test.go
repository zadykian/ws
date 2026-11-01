package sshcfg_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/sshcfg"
)

func TestCheckKeyFile(t *testing.T) {
	tests := []struct{ name, text, want string }{
		{"ed25519 with a comment", "ssh-ed25519 AAAAC3Nza me@mac\n", ""},
		{"a P-256 key", "\necdsa-sha2-nistp256 AAAAE2Vj==\n", ""},
		{"a certificate", "ssh-ed25519-cert-v01@openssh.com AAAA\n", "is a certificate"},
		{"a type that ws lacks", "ssh-mldsa65 AAAA\n", "which ws's OpenSSH lacks"},
		{"two keys", "ssh-ed25519 AAAA\nssh-ed25519 BBBB\n", "holds no public key alone"},
		{"a private key", "-----BEGIN OPENSSH PRIVATE KEY-----\n", "holds no public key alone"},
	}
	for _, tt := range tests {
		err := sshcfg.CheckKeyFile("k.pub", tt.text)
		switch {
		case tt.want == "" && err != nil, tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)):
			t.Errorf("%s: err = %v, want %q", tt.name, err, tt.want)
		}
	}
}
