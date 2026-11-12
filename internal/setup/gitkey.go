package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/internal/sshcfg"
)

// gitKey is the signing key: its .pub, and the key's type and base64. Text is what ws's copy of
// its .pub holds, for a key that Secretive's agent lists.
type gitKey struct{ Pub, Public, Text string }

// key finds the signing key, WS_GIT_SIGNING_PUB_KEY with its ~/ the home. It makes none, and stops
// where mac.conf names none, or an empty one, with the line that would name each P-256 key of
// Secretive's agent. A key that the agent lists needs no file, as macOS 27 denies ws Secretive's
// (docs/mac.md): git names ws's copy. Else it stops where the .pub is missing or unreadable, holds
// more than a public key's line, as a private key does, or ssh-keygen can't read it.
func (g *Git) key(ctx context.Context) (gitKey, error) {
	if g.Conf.GitSigningPubKey == "" {
		// An agent that does not answer leaves a stop with no line to add.
		keys, _, _ := SecretiveKeys(ctx, g.M, g.Home)
		return gitKey{}, unnamed("WS_GIT_SIGNING_PUB_KEY", "git's signing key", keys)
	}
	pub := sshcfg.Expand(g.Conf.GitSigningPubKey, g.Home)
	if sshcfg.SecretiveKey(pub, g.Home) {
		// An agent that does not answer leaves the .pub to read, as an older macOS allows.
		keys, _, _ := SecretiveKeys(ctx, g.M, g.Home)
		if k, ok := held(keys, pub, g.Home); ok {
			key, err := sshcfg.ParsePub(k.Line)
			if err != nil {
				return gitKey{}, refused(pub, "WS_GIT_SIGNING_PUB_KEY", err)
			}
			return gitKey{Pub: sshcfg.Expand(k.Pub(), g.Home), Public: key.String(), Text: k.Line + "\n"}, nil
		}
	}
	text, err := g.M.ReadFile(pub)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return gitKey{}, fmt.Errorf("%s, mac.conf's WS_GIT_SIGNING_PUB_KEY, is missing: make the key first, "+
			"as docs/mac-git.md says", pub)
	case errors.Is(err, fs.ErrPermission) && sshcfg.SecretiveKey(pub, g.Home):
		return gitKey{}, fmt.Errorf("cannot read %s, and Secretive's agent lists no key of it: open Secretive, "+
			"or name another in mac.conf's WS_GIT_SIGNING_PUB_KEY", pub)
	case err != nil:
		return gitKey{}, fmt.Errorf("cannot read %s: make it readable, or name another in "+
			"mac.conf's WS_GIT_SIGNING_PUB_KEY", pub)
	}
	key, err := sshcfg.ParsePub(string(text))
	if err != nil {
		return gitKey{}, refused(pub, "WS_GIT_SIGNING_PUB_KEY", err)
	}
	if _, err := g.M.Run(ctx, "ssh-keygen", "-l", "-f", pub); err != nil {
		return gitKey{}, fmt.Errorf("ssh-keygen cannot read %s: name the .pub of a P-256 or ed25519 key in "+
			"mac.conf's WS_GIT_SIGNING_PUB_KEY", pub)
	}
	return gitKey{Pub: pub, Public: key.String()}, nil
}

// notExist reports whether err says a path is missing, and passes on any other error.
func notExist(err error) (bool, error) {
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	return false, err
}
