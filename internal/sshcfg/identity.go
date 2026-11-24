package sshcfg

import (
	"crypto/md5" //nolint:gosec // Secretive names a key's .pub by its MD5 fingerprint
	"encoding/base64"
	"encoding/hex"
	pathpkg "path"
	"strings"

	"github.com/zadykian/ws/internal/etcws"
)

// Identity is the key that ssh offers ws.
type Identity struct {
	// Pub is the key's .pub as /etc/ws/ssh_config names it: ws's copy from /, or a path with ~/ for the home.
	Pub string
	// Agent is its agent's socket as /etc/ws/ssh_config names it, or empty for a key in a file.
	Agent string
	// Text is what the .pub holds, or would hold as ws's copy.
	Text string
}

// SecretiveData is where Secretive keeps its socket and its public keys, under the home. macOS 27
// denies other apps the files there, with no prompt, but its socket still answers.
const SecretiveData = "Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data"

// KeyCopies is where ws keeps its copies of the .pub of Secretive's keys: root's, 755, and its
// copies 644. They exist as ssh and git read a key's .pub, and the agent signs with the key whatever
// the .pub's path.
const KeyCopies = etcws.Dir + "/secretive"

// KeyCopy reports whether pub is a path of ws's copies: one that holds no dot segment and lies under
// KeyCopies. A path like /etc/ws/secretive/../../x leads out of them.
func KeyCopy(pub string) bool {
	return pathpkg.Clean(pub) == pub && strings.HasPrefix(pub, KeyCopies+"/")
}

// File is where ws keeps ssh's hosts: root's, and read by the Include that heads ~/.ssh/config.
const File = etcws.Dir + "/ssh_config"

// AgentKey is a key that Secretive's agent lists.
type AgentKey struct {
	// Name is its .pub's, as Secretive's own: its MD5 fingerprint in hex.
	Name string
	// Line is the key as ssh-add -L prints it.
	Line string
}

// Pub is the path of ws's copy of the key's .pub, from /.
func (k AgentKey) Pub() string { return KeyCopies + "/" + k.Name }

// AgentKeys returns the keys in the output of ssh-add -L. A certificate, which it lists beside its
// key, is none.
func AgentKeys(list string) []AgentKey {
	var keys []AgentKey
	for _, line := range strings.Split(list, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.HasSuffix(fields[0], "-cert-v01@openssh.com") {
			continue
		}
		blob, err := base64.StdEncoding.DecodeString(fields[1])
		if err != nil {
			continue
		}
		sum := md5.Sum(blob) //nolint:gosec // a file's name, as Secretive's, and no check of the key
		keys = append(keys, AgentKey{hex.EncodeToString(sum[:]) + ".pub", line})
	}
	return keys
}

// SecretiveKey reports whether the path, from /, is the .pub of a key of Secretive's: one in its
// container, under the home, or ws's copy.
func SecretiveKey(pub, home string) bool {
	return strings.HasPrefix(pub, home+"/"+SecretiveData+"/PublicKeys/") || KeyCopy(pub)
}

// Expand returns path with a leading ~/ replaced by the home.
func Expand(path, home string) string {
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		return home + "/" + rest
	}
	return path
}

// NamedIdentity is the key of values.yml's WS_IDENTITY_PUB_KEY and WS_IDENTITY_AGENT. A key of
// Secretive's takes its agent where WS_IDENTITY_AGENT names none.
func NamedIdentity(pub, agent, home string) Identity {
	if agent == "" && SecretiveKey(Expand(pub, home), home) {
		agent = "~/" + SecretiveData + "/socket.ssh"
	}
	return Identity{Pub: pub, Agent: agent}
}
