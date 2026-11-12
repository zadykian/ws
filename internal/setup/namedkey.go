package setup

import (
	"errors"
	"fmt"

	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/sshcfg"
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
