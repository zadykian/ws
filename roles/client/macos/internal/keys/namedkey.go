// Package keys reads the keys that mac.conf names for a use, asks Secretive's agent for its keys,
// and plans ws's copies of their .pub files.
package keys

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// KeyUse is what mac.conf names a key for.
type KeyUse int

// The uses are ws's login key, WS_IDENTITY_PUB_KEY (docs/mac.md), and git's signing key,
// WS_GIT_SIGNING_PUB_KEY (docs/mac-git.md).
const (
	LoginKey KeyUse = iota
	SigningKey
)

// keyUses holds the variable, the words and the page of each use, which every stop names.
var keyUses = [...]struct{ name, words, page string }{
	LoginKey:   {"WS_IDENTITY_PUB_KEY", "ws's login key", "docs/mac.md"},
	SigningKey: {"WS_GIT_SIGNING_PUB_KEY", "git's signing key", "docs/mac-git.md"},
}

// named returns the .pub that mac.conf names for the use, and the agent it names for it.
func (u KeyUse) named(c conf.Conf) (pub, agent string) {
	if u == SigningKey {
		return c.GitSigningPubKey, ""
	}
	return c.IdentityPubKey, c.IdentityAgent
}

// refused is the stop where the .pub of file is no key that ws takes, err being why. The remedy
// names the variable of mac.conf that names the file. A nil err passes.
func (u KeyUse) refused(file string, err error) error {
	name := keyUses[u].name
	var typ sshcfg.TypeError
	switch {
	case err == nil:
		return nil
	case errors.Is(err, sshcfg.ErrCertificate):
		return fmt.Errorf("%s %w: name its key's own .pub in mac.conf's %s", file, err, name)
	case errors.As(err, &typ):
		return fmt.Errorf("%s %w: name the .pub of a P-256 or ed25519 key in mac.conf's %s", file, err, name)
	}
	return fmt.Errorf("%s %w", file, err)
}

// NamedKey is the key that mac.conf names for a use, read and checked.
type NamedKey struct {
	Pub   string           // as mac.conf names it, ~/ kept, or ws's copy for a key the agent holds
	File  string           // Pub with ~/ the home
	Key   sshcfg.PublicKey // its type and base64
	Text  string           // what the .pub holds, or what ws's copy will
	Agent string           // the socket ssh asks: WS_IDENTITY_AGENT, else Secretive's for its key
}

// ReadKey reads the key that mac.conf names for u. It asks Secretive's agent where mac.conf names
// none, or names one of Secretive's, whose key needs no file: macOS 27 denies ws Secretive's
// (docs/mac.md). Each stop names u's variable, and the missing one u's page.
func ReadKey(ctx context.Context, m effect.Machine, c conf.Conf, home string, u KeyUse) (NamedKey, error) {
	name := keyUses[u].name
	pub, agent := u.named(c)
	// An agent that does not answer leaves a named .pub to read, and a stop with no line to add.
	if pub == "" {
		keys, _, _ := SecretiveKeys(ctx, m, home)
		return NamedKey{}, unnamed(u, keys)
	}
	file := sshcfg.Expand(pub, home)
	k := NamedKey{Pub: pub, File: file, Agent: sshcfg.NamedIdentity(pub, agent, home).Agent}
	if sshcfg.SecretiveKey(file, home) {
		keys, _, _ := SecretiveKeys(ctx, m, home)
		if a, ok := held(keys, file, home); ok {
			k.Pub, k.Text = a.Pub(), a.Line+"\n"
			k.File = sshcfg.Expand(k.Pub, home)
			var err error
			k.Key, err = sshcfg.ParsePub(a.Line)
			return k, u.refused(k.File, err)
		}
	}
	text, err := m.ReadFile(file)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return k, fmt.Errorf("%s, mac.conf's %s, is missing: make the key first, as %s says",
			file, name, keyUses[u].page)
	case errors.Is(err, fs.ErrPermission) && sshcfg.SecretiveKey(file, home):
		return k, fmt.Errorf("cannot read %s, and Secretive's agent lists no key of it: open Secretive, "+
			"or name another in mac.conf's %s", file, name)
	case err != nil:
		return k, fmt.Errorf("cannot read %s: make it readable, or name another in mac.conf's %s", file, name)
	}
	if k.Key, err = sshcfg.ParsePub(string(text)); err != nil {
		return k, u.refused(file, err)
	}
	if _, err := m.Run(ctx, "ssh-keygen", "-l", "-f", file); err != nil {
		return k, fmt.Errorf("ssh-keygen cannot read %s: name the .pub of a P-256 or ed25519 key in "+
			"mac.conf's %s", file, name)
	}
	k.Text = string(text)
	return k, nil
}
