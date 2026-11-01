package setup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// RootWrite puts After in root's file, through a new file renamed over it, which --check shows as
// a diff.
type RootWrite struct{ Write }

// RootCopy puts the bytes of the program From in root's file To. No diff shows a program.
type RootCopy struct {
	From, To string
	Data     []byte
	Mode     fs.FileMode
}

func (c RootCopy) String() string { return "would copy " + c.From + " to " + c.To }

// RootRemove removes root's file, which holds Before, and --check shows as a diff.
type RootRemove struct {
	Path   string
	Before []byte
}

func (r RootRemove) String() string { return effect.DiffAs(r.Path, "/dev/null", r.Before, nil) }

// rootFile plans root's file with the bytes and mode: written where missing or different, fixed
// where only its owner or mode differ. A nil change means it is right.
func rootFile(m effect.Machine, path string, mode fs.FileMode, want []byte) (step.Change, error) {
	a, err := m.RootAttrs(path)
	if errors.Is(err, fs.ErrNotExist) {
		return RootWrite{Write{Path: path, After: want, Mode: mode}}, nil
	}
	if err != nil {
		return nil, err
	}
	have, err := m.ReadRootFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	if !bytes.Equal(have, want) {
		return RootWrite{Write{Path: path, Before: have, After: want, Exists: true, Mode: mode}}, nil
	}
	if a.UID != 0 || a.GID != 0 || a.Mode.Perm() != mode {
		return RootFix{path, mode, attrsText(a)}, nil
	}
	return nil, nil
}

// applyRootFile applies the changes of root's files that carry bytes.
func applyRootFile(ctx context.Context, m effect.Machine, out effect.Out, c step.Change) error {
	switch c := c.(type) {
	case RootWrite:
		if err := m.WriteRootFile(c.Path, c.After, c.Mode); err != nil {
			return fmt.Errorf("cannot write %s: %w", c.Path, err)
		}
		out.Say("wrote %s", c.Path)
	case RootCopy:
		if err := m.WriteRootFile(c.To, c.Data, c.Mode); err != nil {
			return fmt.Errorf("cannot copy %s to %s: %w", c.From, c.To, err)
		}
		out.Say("copied %s to %s", c.From, c.To)
	case RootRemove:
		if _, err := m.RunRoot(ctx, "rm", "-f", c.Path); err != nil {
			return fmt.Errorf("cannot remove %s: %w", c.Path, err)
		}
		out.Say("removed %s", c.Path)
	default:
		return applyRoot(ctx, m, c)
	}
	return nil
}
