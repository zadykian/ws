package sshcfg

import (
	"crypto/md5" //nolint:gosec // Secretive names a key's .pub by its MD5 fingerprint
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// Identity is the key that ssh offers ws.
type Identity struct {
	// Pub is the key's .pub as config.d/ws names it, with ~/ for the home.
	Pub string
	// Agent is its agent's socket as config.d/ws names it, or empty for a key in a file.
	Agent string
	// Text is what the .pub holds, or would hold as ws's copy.
	Text string
}

// SecretiveData is where Secretive keeps its socket and its public keys, under the home. macOS 27
// denies other apps the files there, with no prompt, but its socket still answers.
const SecretiveData = "Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data"

// KeyCopies is where ws keeps its copies of the .pub of Secretive's keys, under the home. They
// exist as ssh and git read a key's .pub, and the agent signs with the key whatever the .pub's
// path.
const KeyCopies = ".config/ws/secretive"

// AgentKey is a key that Secretive's agent lists.
type AgentKey struct {
	// Name is its .pub's, as Secretive's own: its MD5 fingerprint in hex.
	Name string
	// Line is the key as ssh-add -L prints it.
	Line string
}

// Pub is the path of ws's copy of the key's .pub, with ~/ for the home.
func (k AgentKey) Pub() string { return "~/" + KeyCopies + "/" + k.Name }

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

// SecretiveKey reports whether the path, from the home, is the .pub of a key of Secretive's: one in
// its container, or ws's copy.
func SecretiveKey(pub, home string) bool {
	return strings.HasPrefix(pub, home+"/"+SecretiveData+"/PublicKeys/") ||
		strings.HasPrefix(pub, home+"/"+KeyCopies+"/")
}

// Expand returns path with a leading ~/ replaced by the home.
func Expand(path, home string) string {
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		return home + "/" + rest
	}
	return path
}

// NamedIdentity is the key of mac.conf's WS_IDENTITY_PUB_KEY and WS_IDENTITY_AGENT. A key of
// Secretive's takes its agent where WS_IDENTITY_AGENT names none.
func NamedIdentity(pub, agent, home string) Identity {
	if agent == "" && SecretiveKey(Expand(pub, home), home) {
		agent = "~/" + SecretiveData + "/socket.ssh"
	}
	return Identity{Pub: pub, Agent: agent}
}
