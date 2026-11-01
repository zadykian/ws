package jetbrains

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/host"
)

// Stale is a path under the home that a run left, and is removed as the user.
type Stale struct {
	S    Site
	Path string
}

type removeStale struct{ s Stale }

func (c removeStale) String() string { return "remove " + c.s.Path }

func (c removeStale) Apply(ctx context.Context) error {
	_, err := c.s.S.AsUser(ctx, "rm", "-rf", "--", c.s.Path)
	return err
}

// Plan plans the removal of a path that exists.
func (s Stale) Plan(context.Context) ([]host.Change, error) {
	if _, err := s.S.M.Stat(s.Path); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return []host.Change{removeStale{s}}, nil
}
