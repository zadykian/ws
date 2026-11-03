package shell

import (
	"context"
	"errors"
	"io/fs"
	"path"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// signingKey makes the SSH key that signs commits, once, with no passphrase: tools and claude
// commit unattended, and the key only signs. docs/shell.md says how to revoke it, under Signing.
// ssh-keygen refuses a path whose directory is missing, as a container's ~/.ssh may be, so the
// directory comes first. The key is made as the user, who then owns it.
func signingKey(m effect.Machine, a account, key string) []host.Resource {
	return []host.Resource{
		host.Dir{M: m, Path: path.Dir(key), Mode: 0o700, UID: a.UID, GID: a.GID},
		keygen{M: m, User: a.User, Path: key},
	}
}

type keygen struct {
	M    effect.Machine
	User host.User
	Path string
}

type makeKey struct{ k keygen }

func (c makeKey) String() string { return "make the SSH key " + c.k.Path + " that signs commits" }

func (c makeKey) Apply(ctx context.Context) error {
	return c.k.User.Pass(ctx, c.k.M, "ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", c.k.Path)
}

// Plan makes the key where the private key is missing.
func (k keygen) Plan(context.Context) ([]host.Change, error) {
	_, err := k.M.RootAttrs(k.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return []host.Change{makeKey{k}}, nil
	}
	return nil, err
}
