package host

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/zadykian/ws/effect"
)

// IDs finds the user's uid and primary gid. root's are 0 without a question.
func (u User) IDs(ctx context.Context, m effect.Machine) (uid, gid int, err error) {
	if u.Name == "root" {
		return 0, 0, nil
	}
	ids := [2]int{}
	for i, flag := range []string{"-u", "-g"} {
		out, err := m.Run(ctx, "id", flag, u.Name)
		if err != nil {
			return 0, 0, fmt.Errorf("find user %s: %w", u.Name, err)
		}
		if ids[i], err = strconv.Atoi(strings.TrimSpace(string(out))); err != nil {
			return 0, 0, fmt.Errorf("id %s %s: %w", flag, u.Name, err)
		}
	}
	return ids[0], ids[1], nil
}

// Owned is a step whose parts are made once the user's uid and gid are found, as they come from
// the machine and a dry run on a new one must still show them.
func (u User) Owned(m effect.Machine, title, after string, parts func(uid, gid int) ([]Resource, error)) Lazy {
	return Lazy{Title: title, After: after, Parts: func(ctx context.Context) ([]Resource, error) {
		uid, gid, err := u.IDs(ctx, m)
		if err != nil {
			return nil, err
		}
		return parts(uid, gid)
	}}
}
