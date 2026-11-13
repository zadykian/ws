package sshcfg

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

// Fingerprint is the key's SHA256 fingerprint, as ssh-keygen -l prints it, or empty for a line
// that holds no key.
func (k AgentKey) Fingerprint() string {
	fields := strings.Fields(k.Line)
	if len(fields) < 2 {
		return ""
	}
	blob, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(blob)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:])
}

// Comment is what follows the key in its line, which Secretive's agent gives as the key's name.
func (k AgentKey) Comment() string {
	fields := strings.Fields(k.Line)
	if len(fields) < 3 {
		return ""
	}
	return strings.Join(fields[2:], " ")
}

// Label names the key to the user by its comment and fingerprint, as Secretive's app shows both, or
// by its fingerprint alone where the comment is empty.
func (k AgentKey) Label() string {
	if c := k.Comment(); c != "" {
		return c + " (" + k.Fingerprint() + ")"
	}
	return k.Fingerprint()
}

// P256 reports whether the key is an ECDSA one on P-256, which Secretive makes in the Secure
// Enclave and ws's OpenSSH reads. An ML-DSA key of Secretive's is not.
func (k AgentKey) P256() bool { return strings.HasPrefix(k.Line, "ecdsa-sha2-nistp256 ") }
