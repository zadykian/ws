package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
)

// Guard refuses a tree that anyone but its owner may have changed. mise runs the tree's config, its
// plugin's Lua and its units as root. So Guard refuses a tree that is a link. It refuses a file or
// a directory in the tree that another user owns, or that a group or others may write. A link in
// the tree counts by its target, which must be there and meet the same rule. mise ignores a plugin
// below a directory that others may write. So Guard refuses such a directory above the tree and the
// plugin, and one that another user owns or that is a link. It changes nothing: Plan fails or finds
// no change.
type Guard struct {
	M effect.Machine
	// Dir is the tree, which need not exist yet.
	Dir string
	// Above holds the directories above the tree and the plugin. One that is missing passes, as a
	// Mac may lack it: setup makes it root's, at 755, before it writes below.
	Above []string
	// Owner is the uid that owns the tree and the directories above it: root's.
	Owner int
	// Holds names a file the tree must hold, as a tree that runs as it stands. Empty asks for none.
	Holds string
}

// Plan fails at the first path the guard refuses, and names it.
func (g Guard) Plan(context.Context) ([]resource.Change, error) {
	for _, dir := range g.Above {
		info, err := g.M.Stat(dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.Link != "" {
			return nil, fmt.Errorf("%s is a link: a directory above the plugin ws must be root's own", dir)
		}
		if !g.alone(info) {
			return nil, fmt.Errorf("%s is not root's alone, and mise ignores the plugin ws below it", dir)
		}
	}
	info, err := g.M.Stat(g.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, g.holds()
	}
	if err != nil {
		return nil, err
	}
	if info.Link != "" {
		return nil, fmt.Errorf("%s is a link: the tree that runs as root must be root's own", g.Dir)
	}
	if err := g.walk(g.Dir, info); err != nil {
		return nil, err
	}
	return nil, g.holds()
}

// holds fails where the tree lacks the file that Holds names.
func (g Guard) holds() error {
	if g.Holds == "" {
		return nil
	}
	if _, err := g.M.Stat(path.Join(g.Dir, g.Holds)); err != nil {
		return fmt.Errorf("--in-place is set, and %s holds no %s", g.Dir, g.Holds)
	}
	return nil
}

// alone says whether the owner holds the path alone: no other user owns it, and no group or others
// may write it.
func (g Guard) alone(info effect.Info) bool {
	return info.UID == g.Owner && info.Mode.Perm()&0o022 == 0
}

// walk checks the path and, for a directory, all below it. A link in it counts by its target, whose
// owner and mode Stat gives, as a reader of the link changes the target. The walk goes no further
// into a link: the tree's own links point inside it, where the walk reaches them.
func (g Guard) walk(p string, info effect.Info) error {
	if !g.alone(info) {
		return fmt.Errorf("%s is not root's alone, and holds the config that runs as root:\n%s is of uid %d, with mode %o",
			g.Dir, p, info.UID, info.Mode.Perm())
	}
	if !info.Mode.IsDir() {
		return nil
	}
	names, err := g.M.ReadDir(p)
	if err != nil {
		return err
	}
	for _, name := range names {
		child := path.Join(p, name)
		found, err := g.M.Stat(child)
		if err != nil {
			return err
		}
		if found.Link != "" {
			if err := g.target(child, found); err != nil {
				return err
			}
			continue
		}
		if err := g.walk(child, found); err != nil {
			return err
		}
	}
	return nil
}

// target refuses a link whose target is missing, as another user could make it, or is not the
// owner's alone.
func (g Guard) target(p string, info effect.Info) error {
	if info.Mode == 0 {
		return fmt.Errorf("%s holds the config that runs as root:\n%s links to %s, which is missing", g.Dir, p, info.Link)
	}
	if !g.alone(info) {
		return fmt.Errorf("%s is not root's alone, and holds the config that runs as root:\n"+
			"%s links to %s, which is of uid %d, with mode %o", g.Dir, p, info.Link, info.UID, info.Mode.Perm())
	}
	return nil
}
