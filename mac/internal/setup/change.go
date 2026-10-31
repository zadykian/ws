// Package setup holds the steps of ws setup, each setting up one part of the Mac.
package setup

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// Write puts After in the user's file with Mode. Before is what the file holds, if it exists.
type Write struct {
	Path          string
	Before, After []byte
	Exists        bool
	Mode          fs.FileMode
}

// String is the diff that --check prints. A file that is new comes from /dev/null.
func (w Write) String() string {
	from := w.Path
	if !w.Exists {
		from = "/dev/null"
	}
	return effect.DiffAs(from, w.Path, w.Before, w.After)
}

// MakeDir makes the user's directory.
type MakeDir struct {
	Path string
	Mode fs.FileMode
}

func (d MakeDir) String() string { return fmt.Sprintf("would make %s, mode %o", d.Path, d.Mode) }

// SetMode sets the mode of the user's file or directory.
type SetMode struct {
	Path string
	Mode fs.FileMode
}

func (s SetMode) String() string { return fmt.Sprintf("would set %s's mode to %o", s.Path, s.Mode) }

// Backup keeps the file as To.
type Backup struct{ Path, To string }

func (b Backup) String() string { return fmt.Sprintf("would keep %s as %s", b.Path, b.To) }

// Apply applies one change of the user's files, and says what it did.
func Apply(ctx context.Context, m effect.Machine, out effect.Out, c step.Change) error {
	switch c := c.(type) {
	case Write:
		if err := m.WriteFile(c.Path, c.After, c.Mode); err != nil {
			return fmt.Errorf("cannot write %s: %w", c.Path, err)
		}
		out.Say("wrote %s", c.Path)
	case MakeDir:
		if err := m.Mkdir(c.Path, c.Mode); err != nil {
			return fmt.Errorf("cannot make %s: %w", c.Path, err)
		}
	case SetMode:
		if err := m.Chmod(c.Path, c.Mode); err != nil {
			return fmt.Errorf("cannot set %s's mode: %w", c.Path, err)
		}
	case Backup:
		if _, err := m.Run(ctx, "cp", "-p", c.Path, c.To); err != nil {
			return fmt.Errorf("cannot keep %s as %s: %w", c.Path, c.To, err)
		}
		out.Say("kept the old %s as %s", c.Path, c.To)
	case Run:
		if err := m.Pass(ctx, c.Root, c.Name, c.Args...); err != nil {
			return fmt.Errorf("%s failed: %w", c.line(), err)
		}
	default:
		return applyRootFile(ctx, m, out, c)
	}
	return nil
}
