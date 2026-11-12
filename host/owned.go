package host

import (
	"context"

	"github.com/zadykian/ws/effect"
)

// IDs finds the user's uid and primary gid. root's are 0 without a question.
func (u User) IDs(ctx context.Context, m effect.Machine) (uid, gid int, err error) {
	a, err := u.Lookup(ctx, m)
	return a.UID, a.GID, err
}

// Owned is a step whose parts are made once the user's uid and gid are found.
func (u User) Owned(m effect.Machine, title, after string, parts func(uid, gid int) ([]Resource, error)) Lazy {
	return u.WithAccount(m, title, after, func(a Account) ([]Resource, error) {
		return parts(a.UID, a.GID)
	})
}
