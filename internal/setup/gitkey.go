package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/internal/gitcfg"
	"github.com/zadykian/ws/internal/sshcfg"
)

// gitKey is the signing key: its .pub, and the key's type and base64, which are empty before the
// key is made. Text is what ws's copy of its .pub holds, for a key that Secretive's agent lists.
type gitKey struct {
	Pub, Public, Text string
	Make              bool
}

// key finds the signing key, WS_GIT_SIGNING_PUB_KEY with its ~/ the home. Without it, the key is
// ~/.ssh/id_ed25519_signing, made once, as the shell role does on ws. No other key is made, and no
// .pub replaced. A key that Secretive's agent lists needs no file, as macOS 27 denies ws Secretive's
// (docs/mac.md): git names ws's copy. Else it stops where the .pub is missing or unreadable, holds
// more than a public key's line, as a private key does, or ssh-keygen can't read it.
func (g *Git) key(ctx context.Context) (gitKey, error) {
	pub := sshcfg.Expand(g.Conf.GitSigningPubKey, g.Home)
	if g.Conf.GitSigningPubKey == "" {
		pub = g.Home + "/.ssh/id_ed25519_signing.pub"
		_, err := g.M.Stat(strings.TrimSuffix(pub, ".pub"))
		if missing, err := notExist(err); err != nil {
			return gitKey{}, err
		} else if missing {
			if _, err := g.M.Stat(pub); err == nil {
				return gitKey{}, fmt.Errorf("%s is there without its private key: name it in mac.conf's "+
					"WS_GIT_SIGNING_PUB_KEY where an agent holds the key, or remove it", pub)
			}
			return gitKey{Pub: pub, Make: true}, nil
		}
	}
	if sshcfg.SecretiveKey(pub, g.Home) {
		// An agent that does not answer leaves the .pub to read, as an older macOS allows.
		keys, _, _ := SecretiveKeys(ctx, g.M, g.Home)
		if k, ok := held(keys, pub, g.Home); ok {
			return gitKey{Pub: sshcfg.Expand(k.Pub(), g.Home), Public: gitcfg.PublicKey(k.Line), Text: k.Line + "\n"}, nil
		}
	}
	text, err := g.M.ReadFile(pub)
	if missing, rerr := notExist(err); rerr != nil {
		return gitKey{}, fmt.Errorf("cannot read %s: make it readable, or name another .pub in "+
			"mac.conf's WS_GIT_SIGNING_PUB_KEY", pub)
	} else if missing && g.Conf.GitSigningPubKey == "" {
		return gitKey{}, fmt.Errorf("%s is missing beside its key: ssh-keygen -y -f %s >%s writes it",
			pub, strings.TrimSuffix(pub, ".pub"), pub)
	} else if missing {
		return gitKey{}, fmt.Errorf("%s, mac.conf's WS_GIT_SIGNING_PUB_KEY, is missing: make the key first, "+
			"as docs/mac-git.md says", pub)
	}
	public := gitcfg.PublicKey(string(text))
	if public == "" {
		return gitKey{}, fmt.Errorf("%s holds no public key alone, as ssh-keygen -y prints one", pub)
	}
	if _, err := g.M.Run(ctx, "ssh-keygen", "-l", "-f", pub); err != nil {
		return gitKey{}, fmt.Errorf("ssh-keygen cannot read %s: git signs with a key the Mac's ssh-keygen "+
			"knows, as ed25519 or Secretive's P-256", pub)
	}
	return gitKey{Pub: pub, Public: public}, nil
}

// notExist reports whether err says a path is missing, and passes on any other error.
func notExist(err error) (bool, error) {
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	return false, err
}
