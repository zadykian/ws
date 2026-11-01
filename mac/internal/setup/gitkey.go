package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/mac/internal/gitcfg"
	"github.com/zadykian/ws/mac/internal/sshcfg"
)

// gitKey is the signing key: its .pub, and the key's type and base64, which are empty before the
// key is made.
type gitKey struct {
	Pub, Public string
	Make        bool
}

// key finds the signing key, WS_GIT_SIGNING_KEY with its ~/ the home. Without it, the key is
// ~/.ssh/id_ed25519_signing, made once, as the shell role does on ws. No other key is made, and no
// .pub replaced. It stops where the .pub is missing or unreadable, holds more than a public key's
// line, as a private key does, or ssh-keygen can't read it.
func (g *Git) key(ctx context.Context) (gitKey, error) {
	pub := sshcfg.Expand(g.Conf.GitSigningKey, g.Home)
	if g.Conf.GitSigningKey == "" {
		pub = g.Home + "/.ssh/id_ed25519_signing.pub"
		_, err := g.M.Stat(strings.TrimSuffix(pub, ".pub"))
		if missing, err := notExist(err); err != nil {
			return gitKey{}, err
		} else if missing {
			if _, err := g.M.Stat(pub); err == nil {
				return gitKey{}, fmt.Errorf("%s is there without its private key: name it in mac.conf's "+
					"WS_GIT_SIGNING_KEY where an agent holds the key, or remove it", pub)
			}
			return gitKey{Pub: pub, Make: true}, nil
		}
	}
	text, err := g.M.ReadFile(pub)
	if missing, rerr := notExist(err); rerr != nil {
		return gitKey{}, fmt.Errorf("cannot read %s: make it readable, or name another .pub in "+
			"mac.conf's WS_GIT_SIGNING_KEY", pub)
	} else if missing && g.Conf.GitSigningKey == "" {
		return gitKey{}, fmt.Errorf("%s is missing beside its key: ssh-keygen -y -f %s >%s writes it",
			pub, strings.TrimSuffix(pub, ".pub"), pub)
	} else if missing {
		return gitKey{}, fmt.Errorf("%s, mac.conf's WS_GIT_SIGNING_KEY, is missing: make the key first, "+
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
