package sshcfg

import (
	"errors"
	"regexp"
	"strings"
)

var base64Key = regexp.MustCompile(`^[A-Za-z0-9+/]+=*$`)

// keyTypes are the types of ws's OpenSSH 10.2 (ssh -Q key there).
var keyTypes = map[string]bool{
	"ssh-ed25519": true, "ssh-rsa": true, "sk-ssh-ed25519@openssh.com": true,
	"sk-ecdsa-sha2-nistp256@openssh.com": true,
	"ecdsa-sha2-nistp256":                true, "ecdsa-sha2-nistp384": true, "ecdsa-sha2-nistp521": true,
}

// PublicKey is the type and base64 of the one key that a .pub holds.
type PublicKey struct{ Type, Base64 string }

// String is the key as allowed_signers and GitHub take it: the type, a space and the base64.
func (k PublicKey) String() string { return k.Type + " " + k.Base64 }

// ErrCertificate and ErrNotAKey end a sentence that names the file. They are why ParsePub refuses a
// text.
var (
	ErrCertificate = errors.New("is a certificate")
	ErrNotAKey     = errors.New("holds no public key alone, as ssh-keygen -y prints one")
)

// TypeError is a key of a type that ws's OpenSSH lacks, as an ML-DSA key of Secretive's. Its name
// starts with ssh-, ecdsa-sha2- or sk-.
type TypeError struct{ Type string }

func (e TypeError) Error() string {
	return "holds a key of type " + e.Type + ", which ws's OpenSSH lacks"
}

// ParsePub returns the key of a .pub's text: one line that is not blank, of one of the seven types
// of ws's OpenSSH, with a base64 data. A private key, a text of two keys, a certificate or a type
// that ws lacks fails, with ErrNotAKey, ErrCertificate or a TypeError.
func ParsePub(text string) (PublicKey, error) {
	var fields []string
	lines := 0
	for _, line := range strings.Split(text, "\n") {
		if f := strings.Fields(line); len(f) > 0 {
			fields = f
			lines++
		}
	}
	if lines != 1 || len(fields) < 2 || !base64Key.MatchString(fields[1]) {
		return PublicKey{}, ErrNotAKey
	}
	switch typ := fields[0]; {
	case keyTypes[typ]:
		return PublicKey{typ, fields[1]}, nil
	case strings.HasSuffix(typ, "-cert-v01@openssh.com"):
		return PublicKey{}, ErrCertificate
	case strings.HasPrefix(typ, "ssh-") || strings.HasPrefix(typ, "ecdsa-sha2-") || strings.HasPrefix(typ, "sk-"):
		return PublicKey{}, TypeError{typ}
	}
	return PublicKey{}, ErrNotAKey
}
