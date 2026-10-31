package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// RootDir makes a root's directory.
type RootDir struct {
	Path string
	Mode fs.FileMode
}

func (d RootDir) String() string {
	return fmt.Sprintf("would make %s, root's, with mode %o", d.Path, d.Mode)
}

// RootFix gives root's file or directory root and Mode. Was is what it has now, as 501:20 644.
type RootFix struct {
	Path string
	Mode fs.FileMode
	Was  string
}

func (f RootFix) String() string {
	return fmt.Sprintf("would make %s root's, with mode %o, rather than %s", f.Path, f.Mode, f.Was)
}

func attrsText(a effect.Attrs) string { return fmt.Sprintf("%d:%d %o", a.UID, a.GID, a.Mode.Perm()) }

// rootDir plans root's directory with mode: made where missing, fixed where its owner or mode differ.
func rootDir(m effect.Machine, path string, mode fs.FileMode) (step.Change, error) {
	a, err := m.RootAttrs(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return RootDir{path, mode}, nil
	case err != nil:
		return nil, err
	case a.UID != 0 || a.GID != 0 || a.Mode.Perm() != mode:
		return RootFix{path, mode, attrsText(a)}, nil
	}
	return nil, nil
}

// applyRoot applies a change of root's files.
func applyRoot(ctx context.Context, m effect.Machine, c step.Change) error {
	switch c := c.(type) {
	case RootDir:
		if _, err := m.RunRoot(ctx, "mkdir", "-m", fmt.Sprintf("%o", c.Mode), c.Path); err != nil {
			return fmt.Errorf("cannot make %s: %w", c.Path, err)
		}
		if _, err := m.RunRoot(ctx, "chown", "0:0", c.Path); err != nil {
			return fmt.Errorf("cannot make %s: %w", c.Path, err)
		}
	case RootFix:
		_, err := m.RunRoot(ctx, "chown", "0:0", c.Path)
		if err == nil {
			_, err = m.RunRoot(ctx, "chmod", fmt.Sprintf("%o", c.Mode), c.Path)
		}
		if err != nil {
			return fmt.Errorf("cannot set %s's owner and mode: %w", c.Path, err)
		}
	default:
		return fmt.Errorf("unknown change %T", c)
	}
	return nil
}
