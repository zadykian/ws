package git

import (
	"context"
	"errors"
	"io/fs"
	"path"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/account"
	"github.com/zadykian/ws/roles/server/ubuntu/shell/internal/plan"
)

// SigningKey makes the SSH key that signs commits, once, with no passphrase: tools and claude
// commit unattended, and the key only signs. docs/shell.md says how to revoke it, under Signing.
// ssh-keygen refuses a path whose directory is missing, as a container's ~/.ssh may be, so the
// directory comes first. A key in the home itself needs none. The user makes the key, and owns it.
func SigningKey(m effect.Machine, a account.Account, key string) []host.Resource {
	gen := keygen{M: m, User: a.User, Path: key}
	if dir := plan.Below(a, path.Dir(key)); dir != "" {
		return []host.Resource{a.Dir(m, dir, 0o700), gen}
	}
	return []host.Resource{gen}
}

type keygen struct {
	M    effect.Machine
	User account.User
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
