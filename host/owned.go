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

// Owned is a step whose parts are made once the user's uid and gid are found. They come from
// the machine, and a dry run on a new one must still show them.
func (u User) Owned(m effect.Machine, title, after string, parts func(uid, gid int) ([]Resource, error)) Lazy {
	return Lazy{Title: title, After: after, Parts: func(ctx context.Context) ([]Resource, error) {
		uid, gid, err := u.IDs(ctx, m)
		if err != nil {
			return nil, err
		}
		return parts(uid, gid)
	}}
}
