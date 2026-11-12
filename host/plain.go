package host

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/effect"
)

// Plain refuses a link at any of Paths, or at a directory between Home and them. The user owns its
// home, so a link there would send the root ws runs as, with its reads, chown, chmod and backup, to
// whatever the link names. It looks as it plans, and names the link. A link made while the step
// applies is caught by effect.OSMachine's Confine, which fails the write itself (docs/security.md).
type Plain struct {
	M     effect.Machine
	Home  string
	Paths []string
}

// Plan plans nothing, and fails on the first link.
func (p Plain) Plan(context.Context) ([]Change, error) {
	return nil, RefuseLinks(p.M, p.Home, p.Paths...)
}

// RefuseLinks fails on the first link at any of paths, or at a directory between home and them,
// for a step that learns its paths as it plans. A path outside home is checked alone. It fails
// where Stat fails but for a missing path, as a link may lie there unseen.
func RefuseLinks(m effect.Machine, home string, paths ...string) error {
	for _, target := range paths {
		for _, at := range chain(home, target) {
			info, err := m.Stat(at)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("look for a link at %s: %w", at, err)
			}
			if info.Link != "" {
				return fmt.Errorf("%s is a link to %s: replace it with the file itself", at, info.Link)
			}
		}
	}
	return nil
}
