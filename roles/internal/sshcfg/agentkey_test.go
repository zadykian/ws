package sshcfg_test

import (
	"testing"

	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// p256 is a P-256 key's line, whose fingerprint ssh-keygen -l gives as fp.
const (
	p256 = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHE48m9xL7l3Xy/I73eBgYXJJRF" +
		"Y7sTC8AIh5hXBwm7zjXQLZ15NOX8V62nMiAHs8Gi7f6WuMH8T8d2/oy/3di4="
	fp = "SHA256:REPwNMsHfqZbGTbqNcni71w8U2mX89lYsYzakyx+Uk4"
)

func TestAgentKeyLabel(t *testing.T) {
	for line, want := range map[string]string{
		p256 + " my key": "my key (" + fp + ")",
		p256:             fp,
		"ssh-ed25519":    "",
	} {
		if got := (sshcfg.AgentKey{Line: line}).Label(); got != want {
			t.Errorf("Label of %q = %q, want %q", line, got, want)
		}
	}
}

func TestAgentKeyP256(t *testing.T) {
	if !(sshcfg.AgentKey{Line: p256}).P256() || (sshcfg.AgentKey{Line: "ssh-mldsa65 AAAA x"}).P256() {
		t.Error("P256 took an ML-DSA key, or not a P-256 one")
	}
}
