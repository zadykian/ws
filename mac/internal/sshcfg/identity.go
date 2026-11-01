package sshcfg

import (
	"fmt"
	"regexp"
	"strings"
)

// Identity is the key that ssh offers ws.
type Identity struct {
	// Pub is the key's .pub as config.d/ws names it, with ~/ for the home.
	Pub string
	// Agent is its agent's socket as config.d/ws names it, or empty for a key in a file.
	Agent string
	// Check is whether the .pub must exist and hold a key: all but the default file key.
	Check bool
}

// DefaultPub is the file key's .pub, which ssh offers ws unless mac.conf or Secretive says other.
const DefaultPub = "~/.ssh/id_ed25519.pub"

// SecretiveData is where Secretive keeps its socket and its public keys, under the home.
const SecretiveData = "Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data"

var plainName = regexp.MustCompile(`^[A-Za-z0-9._@+-]+$`)

// Expand returns path with a leading ~/ replaced by the home.
func Expand(path, home string) string {
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		return home + "/" + rest
	}
	return path
}

// PickIdentity picks the key from mac.conf's WS_IDENTITY and WS_IDENTITY_AGENT, where set. Else,
// where Secretive's socket is there, it is the one .pub in publicKeys, the names in Secretive's
// PublicKeys. git's signing key, which signs alone, does not count, nor a certificate, -cert.pub,
// which Secretive kept there before 4.0, nor a dotfile, nor a name ssh_config would misread, as
// Secretive's own are hex. Else it is the file key. A key in PublicKeys takes Secretive's agent. It fails where Secretive
// holds several such keys.
func PickIdentity(identity, agent, signingKey, home string, socket bool, publicKeys []string) (Identity, error) {
	secretive := home + "/" + SecretiveData
	pub := identity
	if pub == "" && socket {
		skip := Expand(signingKey, home)
		var keys []string
		for _, name := range publicKeys {
			full := secretive + "/PublicKeys/" + name
			if !strings.HasSuffix(name, ".pub") || full == skip || strings.HasSuffix(name, "-cert.pub") ||
				strings.HasPrefix(name, ".") || !plainName.MatchString(name) {
				continue
			}
			keys = append(keys, full)
			pub = "~/" + strings.TrimPrefix(full, home+"/")
		}
		if len(keys) > 1 {
			//nolint:staticcheck // Secretive is a name, and mac-identity-checks.sh expects this text
			return Identity{}, fmt.Errorf("Secretive's agent holds several keys, %s: name the one for ws in "+
				"mac.conf's WS_IDENTITY", strings.Join(keys, ", "))
		}
	}
	if pub == "" {
		pub = DefaultPub
	}
	if pub+agent == DefaultPub {
		return Identity{Pub: pub}, nil
	}
	if agent == "" && strings.HasPrefix(Expand(pub, home), secretive+"/PublicKeys/") {
		agent = "~/" + strings.TrimPrefix(secretive, home+"/") + "/socket.ssh"
	}
	return Identity{Pub: pub, Agent: agent, Check: true}, nil
}
