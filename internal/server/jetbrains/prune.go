package jetbrains

import (
	"context"
	"errors"
	"io/fs"
	"path"

	"github.com/zadykian/ws/host"
)

// prune removes a build that the latest one replaces. It does so under the lock: a DIR.part that
// another run unpacks into has its build's name by the time this run holds the lock, and so stays.
type prune struct {
	s    Site
	path string
}

func (c prune) String() string { return "remove " + c.path + ", which the latest build replaces" }

func (c prune) Apply(ctx context.Context) error {
	return c.s.Lock(c.s.Dist, func() error {
		if _, err := c.s.M.Stat(c.path); errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		_, err := c.s.M.Run(ctx, "rm", "-rf", "--", c.path)
		return err
	})
}

// others plans the removal of the other builds of the backend's product, Gateway's own downloads
// too; other products' stay. A build that a process runs from, such as a backend Gateway started,
// stays until a run finds none.
func others(ctx context.Context, s Site, b Build) ([]host.Change, error) {
	names, err := s.M.ReadDir(s.Dist)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil // Gateway's directory is not made yet
	}
	if err != nil {
		return nil, err
	}
	var changes []host.Change
	for _, name := range names {
		dir := path.Join(s.Dist, name)
		if !b.Pattern.MatchString(name) || dir == b.Dir {
			continue
		}
		pids, err := s.running(ctx, dir)
		if err != nil {
			return nil, err
		}
		if len(pids) > 0 {
			s.Out.Say("%s stays, as a process runs from it: the next run that finds none removes it.", name)
			continue
		}
		changes = append(changes, prune{s, dir})
	}
	return changes, nil
}
