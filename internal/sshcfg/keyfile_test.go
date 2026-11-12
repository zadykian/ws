package sshcfg_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/sshcfg"
)

func TestCheckKeyFile(t *testing.T) {
	tests := []struct{ name, text, want string }{
		{"ed25519 with a comment", "ssh-ed25519 AAAAC3Nza me@mac\n", ""},
		{"a P-256 key", "\necdsa-sha2-nistp256 AAAAE2Vj==\n", ""},
		{"a certificate", "ssh-ed25519-cert-v01@openssh.com AAAA\n", "is a certificate: name its key's own .pub in mac.conf's WS_X"},
		{"a type that ws lacks", "ssh-mldsa65 AAAA\n", "which ws's OpenSSH lacks"},
		{"two keys", "ssh-ed25519 AAAA\nssh-ed25519 BBBB\n", "holds no public key alone"},
		{"a private key", "-----BEGIN OPENSSH PRIVATE KEY-----\n", "holds no public key alone"},
	}
	for _, tt := range tests {
		err := sshcfg.CheckKeyFile("k.pub", "WS_X", tt.text)
		switch {
		case tt.want == "" && err != nil, tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)):
			t.Errorf("%s: err = %v, want %q", tt.name, err, tt.want)
		}
	}
}

func TestParsePub(t *testing.T) {
	good := []string{
		"ssh-ed25519", "ssh-rsa", "sk-ssh-ed25519@openssh.com", "sk-ecdsa-sha2-nistp256@openssh.com",
		"ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521",
	}
	for _, typ := range good {
		key, err := sshcfg.ParsePub(typ + " AAAA+/==\n")
		if err != nil || key.Type != typ || key.Base64 != "AAAA+/==" || key.String() != typ+" AAAA+/==" {
			t.Errorf("%s: ParsePub = %+v, %v", typ, key, err)
		}
	}
	if key, err := sshcfg.ParsePub("\n  ssh-ed25519 AAAAC3Nza\tme@mac\n\n"); err != nil || key.String() != "ssh-ed25519 AAAAC3Nza" {
		t.Errorf("a comment and blank lines: ParsePub = %+v, %v", key, err)
	}
	notKey := map[string]string{
		"two keys":    "ssh-ed25519 AAAA\nssh-ed25519 BBBB\n",
		"private key": "-----BEGIN OPENSSH PRIVATE KEY-----\nAAAA\n-----END OPENSSH PRIVATE KEY-----\n",
		"a type":      "ssh-ed25519\n",
		"bad base64":  "ssh-ed25519 AA*A\n",
		"other type":  "rsa AAAA\n",
		"empty":       "",
	}
	for name, text := range notKey {
		if _, err := sshcfg.ParsePub(text); !errors.Is(err, sshcfg.ErrNotAKey) {
			t.Errorf("%s: err = %v, want ErrNotAKey", name, err)
		}
	}
	if _, err := sshcfg.ParsePub("ssh-ed25519-cert-v01@openssh.com AAAA\n"); !errors.Is(err, sshcfg.ErrCertificate) {
		t.Errorf("a certificate: err = %v", err)
	}
	_, err := sshcfg.ParsePub("ssh-mldsa-65 AAAA\n")
	var typ sshcfg.TypeError
	if !errors.As(err, &typ) || typ.Type != "ssh-mldsa-65" {
		t.Errorf("a type that ws lacks: err = %v", err)
	}
}
