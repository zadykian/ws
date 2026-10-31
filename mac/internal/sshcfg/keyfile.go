package sshcfg

import (
	"fmt"
	"regexp"
	"strings"
)

var base64Key = regexp.MustCompile(`^[A-Za-z0-9+/]+=*$`)

// CheckKeyFile fails where the text of the .pub file holds other than one public key of a type that
// ws's OpenSSH 10.2 has (ssh -Q key there): a private key or a certificate does, and so does a type
// as an ML-DSA key of Secretive's.
func CheckKeyFile(file, text string) error {
	var keyType string
	lines := 0
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		lines++
		keyType = ""
		if len(fields) > 1 && base64Key.MatchString(fields[1]) {
			keyType = fields[0]
		}
	}
	if lines != 1 {
		keyType = ""
	}
	switch {
	case keyType == "ssh-ed25519", keyType == "ssh-rsa", keyType == "sk-ssh-ed25519@openssh.com",
		keyType == "sk-ecdsa-sha2-nistp256@openssh.com",
		keyType == "ecdsa-sha2-nistp256", keyType == "ecdsa-sha2-nistp384", keyType == "ecdsa-sha2-nistp521":
		return nil
	case strings.HasSuffix(keyType, "-cert-v01@openssh.com"):
		return fmt.Errorf("%s is a certificate: name its key's own .pub in mac.conf's WS_IDENTITY", file)
	case strings.HasPrefix(keyType, "ssh-") || strings.HasPrefix(keyType, "ecdsa-sha2-") ||
		strings.HasPrefix(keyType, "sk-"):
		return fmt.Errorf("%s holds a key of type %s, which ws's OpenSSH lacks: name the .pub of "+
			"a P-256 or ed25519 key in mac.conf's WS_IDENTITY", file, keyType)
	}
	return fmt.Errorf("%s holds no public key alone, as ssh-keygen -y prints one", file)
}
