package tmux

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
)

// Snap removes tmux's snap. snapd must run to remove a snap, so without systemd, as in CI's
// container, the removal skips and says so.
type Snap struct {
	M   effect.Machine
	Out effect.Out
}

type removeSnap struct{ s Snap }

func (removeSnap) String() string { return "remove the tmux snap" }

func (c removeSnap) Apply(ctx context.Context) error {
	_, err := c.s.M.RunRoot(ctx, "snap", "remove", "tmux")
	return err
}

// Plan lists the snaps, where snapd is installed and runs, and removes tmux's.
func (s Snap) Plan(ctx context.Context) ([]host.Change, error) {
	if !host.Booted(s.M) {
		s.Out.Say("tmux snap: skipped, systemd is not running")
		return nil, nil
	}
	if _, err := s.M.Stat("/usr/bin/snap"); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	list, err := s.M.Run(ctx, "snap", "list")
	if err != nil {
		return nil, err
	}
	if !hasSnap(string(list), "tmux") {
		return nil, nil
	}
	return []host.Change{removeSnap{s}}, nil
}

// hasSnap reports whether the output of snap list names the snap in its first column, below the
// header.
func hasSnap(list, name string) bool {
	lines := strings.Split(list, "\n")
	for _, line := range lines[min(1, len(lines)):] {
		if f := strings.Fields(line); len(f) > 0 && f[0] == name {
			return true
		}
	}
	return false
}
