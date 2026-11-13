// Package change holds the changes that the Mac's roles plan and apply: a write of the user's file, a
// command, a key copy. Each says what it would do, and Apply does it.
package change

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
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

// Excerpt is the diff of what ws adds to a file it keeps the text of. Such a file is the user's, and
// the lines beside ws's may hold a secret. ws replaces its own files whole, and their diff stands.
func (w Write) Excerpt() string {
	if len(w.Before) == 0 || !bytes.Contains(w.After, w.Before) {
		return w.String()
	}
	return effect.DiffAs(w.Path, w.Path, nil, bytes.Replace(w.After, w.Before, nil, 1))
}

// Title names the file, which a new file's first line of diff leaves out.
func (w Write) Title() string {
	if !w.Exists {
		return "would create " + w.Path
	}
	return "would write " + w.Path
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

// Backup keeps a copy of the file at To.
type Backup struct{ Path, To string }

func (b Backup) String() string { return fmt.Sprintf("would keep %s as %s", b.Path, b.To) }

// Remark is a line that --check prints in the form of a change, which a run does not apply.
type Remark string

func (r Remark) String() string { return string(r) }

// Apply applies one change of the user's files, and says what it did.
func Apply(ctx context.Context, m effect.Machine, out effect.Out, c step.Change) error {
	switch c := c.(type) {
	case Remark:
		return nil
	case Write:
		if err := m.WriteFile(c.Path, c.After, c.Mode); err != nil {
			return fmt.Errorf("cannot write %s: %w", c.Path, err)
		}
		out.Say("wrote %s", c.Path)
	case KeyCopy:
		return applyCopy(m, out, c)
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
			return fmt.Errorf("%s failed: %w", c.Line(), err)
		}
	case resource.Change:
		if err := c.Apply(ctx); err != nil {
			return fmt.Errorf("cannot %s: %w", step.Headline(c), err)
		}
		out.Say("%s", step.Headline(c))
	default:
		return fmt.Errorf("unknown change %T", c)
	}
	return nil
}
