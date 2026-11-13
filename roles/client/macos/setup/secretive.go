package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/sshcfg"
)

// ErrAgentDown is why Secretive's keys cannot be listed: its socket stays after Secretive quits, and
// nothing answers there.
var ErrAgentDown = errors.New("Secretive's agent does not answer") //nolint:staticcheck // Secretive is a name

// SecretiveKeys returns the keys that Secretive's agent lists, and whether its socket is there. It
// asks the agent, as macOS 27 denies ws the files of Secretive's container (docs/mac.md). A socket
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
			"mac.conf's WS_IDENTITY_PUB_KEY", ErrAgentDown, socket)
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

// unnamed is the stop where mac.conf names no key for the use. It gives the line that would name
// each P-256 key of keys, the agent's, by its comment and fingerprint. ws's OpenSSH reads no other
// key of Secretive's.
func unnamed(u KeyUse, keys []sshcfg.AgentKey) error {
	name, use := keyUses[u].name, keyUses[u].words
	var lines []string
	for _, k := range keys {
		if k.P256() {
			lines = append(lines, name+"="+k.Pub()+" for "+k.Label())
		}
	}
	if len(lines) == 0 {
		return fmt.Errorf("mac.conf names no %s, %s: run ws client macos wizard to pick or make one in Secretive, or "+
			"name a key's .pub there", name, use)
	}
	return fmt.Errorf("mac.conf names no %s, %s: run ws client macos wizard, or add one of these lines to mac.conf, %s",
		name, use, strings.Join(lines, "; "))
}

// KeyCopy writes ws's copy of the .pub of a key of Secretive's.
type KeyCopy struct{ Path, Text string }

func (c KeyCopy) String() string {
	return "would write " + c.Path + ", ws's copy of a key of Secretive's"
}

// PlanCopy returns the changes that make ws's copy at pub hold text: the directories of the copies,
// ~/.config/ws and ~/.config with them, where missing, then the copy. It returns none for a .pub
// elsewhere and for one with no text from the agent, and no copy where it holds text already.
func PlanCopy(m effect.Machine, pub, text, home string) ([]step.Change, error) {
	if !copied(pub, text, home) {
		return nil, nil
	}
	cs, _, err := copyDirs(m, home).plan()
	if err != nil {
		return nil, err
	}
	return append(cs, copyChange(m, pub, text)...), nil
}

// copyDirs is ws's directory with the directory of ws's copies in it.
func copyDirs(m effect.Machine, home string) userDirs {
	return append(wsDirs(m, home), userDir{M: m, Path: home + "/" + sshcfg.KeyCopies, Mode: 0o755})
}

// copyChange returns the write of ws's copy at pub, or none where it holds text already.
func copyChange(m effect.Machine, pub, text string) []step.Change {
	if old, err := m.ReadFile(pub); err == nil && string(old) == text {
		return nil
	}
	return []step.Change{KeyCopy{pub, text}}
}

// copied says whether ws keeps a copy of the .pub at pub: one in ws's directory of copies, with text
// from the agent.
func copied(pub, text, home string) bool {
	return text != "" && strings.HasPrefix(pub, home+"/"+sshcfg.KeyCopies+"/")
}

// keyCopy reports ws's copy of a .pub, where it keeps one. A run writes the copy where its text
// differs, and leaves its mode.
func keyCopy(m effect.Machine, pub, text, home string) []step.Reporter {
	if !copied(pub, text, home) {
		return nil
	}
	return []step.Reporter{userFile{M: m, Path: pub, Text: []byte(text)}}
}

func applyCopy(m effect.Machine, out effect.Out, c KeyCopy) error {
	if err := m.WriteFile(c.Path, []byte(c.Text), 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", c.Path, err)
	}
	out.Say("wrote %s", c.Path)
	return nil
}
