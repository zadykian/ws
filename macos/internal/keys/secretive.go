package keys

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/etcws"
	"github.com/zadykian/ws/internal/sshcfg"
	"github.com/zadykian/ws/macos/values"
)

// ErrAgentDown is why Secretive's keys cannot be listed: its socket stays after Secretive quits, and
// nothing answers there.
var ErrAgentDown = errors.New("Secretive's agent does not answer") //nolint:staticcheck // Secretive is a name

// SecretiveKeys returns the keys that Secretive's agent lists, and whether its socket is there. It
// asks the agent, as macOS 27 denies ws the files of Secretive's container (macos/README.md). A socket
// with no agent is ErrAgentDown.
func SecretiveKeys(ctx context.Context, m effect.Machine, home string) ([]sshcfg.AgentKey, bool, error) {
	socket := home + "/" + sshcfg.SecretiveData + "/socket.ssh"
	info, err := m.Stat(socket)
	if err != nil || info.Mode&fs.ModeSocket == 0 {
		return nil, false, nil
	}
	out, err := m.Run(ctx, "env", "SSH_AUTH_SOCK="+socket, "ssh-add", "-L")
	// ssh-add exits 1 and says so on its standard output for an agent with no keys.
	if err != nil && !strings.Contains(string(out), "The agent has no identities") {
		return nil, true, fmt.Errorf("%w at %s: open Secretive, then run this again, or name another key in "+
			"values.yml's WS_IDENTITY_PUB_KEY", ErrAgentDown, socket)
	}
	return sshcfg.AgentKeys(string(out)), true, nil
}

// held returns the key among keys whose .pub the path names, in Secretive's container or among ws's
// copies, which share their names.
func held(keys []sshcfg.AgentKey, pub, home string) (sshcfg.AgentKey, bool) {
	if sshcfg.SecretiveKey(pub, home) {
		for _, k := range keys {
			if k.Name == filepath.Base(pub) {
				return k, true
			}
		}
	}
	return sshcfg.AgentKey{}, false
}

// unnamed is the stop where values.yml names no key for the use. It gives the line that would name
// each P-256 key of keys, the agent's, by its comment and fingerprint. ws's OpenSSH reads no other
// key of Secretive's.
func unnamed(u KeyUse, keys []sshcfg.AgentKey) error {
	name, use := keyUses[u].name, keyUses[u].words
	var lines []string
	for _, k := range keys {
		if k.P256() {
			lines = append(lines, values.Line(name, k.Pub())+" for "+k.Label())
		}
	}
	if len(lines) == 0 {
		return fmt.Errorf("values.yml names no %s, %s: make a P-256 key in Secretive, then "+
			"name its .pub there", name, use)
	}
	return fmt.Errorf("values.yml names no %s, %s: add one of these lines to values.yml, %s",
		name, use, strings.Join(lines, "; "))
}

// Copied says whether ws keeps a copy of the .pub at pub: one in ws's directory of copies, with
// text from the agent. A path that leads out of the directory is none.
func Copied(pub, text string) bool {
	return text != "" && sshcfg.KeyCopy(pub)
}

// CopyParts returns /etc/ws, the directory of the copies and ws's copy of the .pub at pub, which
// say their changes as they apply. It returns none for a .pub elsewhere and for one with no text
// from the agent. A run writes the copy through sudo where its text differs.
func CopyParts(m effect.Machine, out effect.Out, pub, text string) []resource.Said {
	if !Copied(pub, text) {
		return nil
	}
	var parts []resource.Said
	for _, r := range etcws.Files(m, pub, []byte(text)) {
		parts = append(parts, resource.Said{Out: out, Part: r})
	}
	return parts
}

// PlanCopy returns the changes that make ws's copy at pub hold text, with the directories of the
// copies where missing. It returns none where the copy holds text already.
func PlanCopy(ctx context.Context, m effect.Machine, out effect.Out, pub, text string) ([]step.Change, error) {
	var cs []step.Change
	for _, part := range CopyParts(m, out, pub, text) {
		var err error
		if cs, err = resource.Append(ctx, cs, part); err != nil {
			return nil, err
		}
	}
	return cs, nil
}

// Copies returns ws's copies of the .pubs of the login and the signing key that c names, which ws
// macos keys writes. Each key that Secretive's agent holds has one, and a key in a file has
// none. A key that both name has one copy.
func Copies(ctx context.Context, m effect.Machine, out effect.Out, c values.Conf, home string) ([]resource.Resource, error) {
	var rs []resource.Resource
	seen := map[string]bool{}
	for _, u := range []KeyUse{LoginKey, SigningKey} {
		k, err := ReadKey(ctx, m, c, home, u)
		if err != nil {
			return nil, err
		}
		if seen[k.File] {
			continue
		}
		seen[k.File] = true
		for _, part := range CopyParts(m, out, k.File, k.Text) {
			rs = append(rs, part)
		}
	}
	return rs, nil
}
