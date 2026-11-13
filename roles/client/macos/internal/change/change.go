// Package change holds the changes that the Mac's roles plan and apply: a write of the user's file, a
// command, a key copy. Each says what it would do, and Apply does it.
package change

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
)

// Remark is a line that --check prints in the form of a change, which a run does not apply.
type Remark string

func (r Remark) String() string { return string(r) }

// Apply applies one change of the user's files, and says what it did.
func Apply(ctx context.Context, m effect.Machine, out effect.Out, c step.Change) error {
	switch c := c.(type) {
	case Remark:
		return nil
	case file.UserWrite:
		if err := m.WriteFile(c.Path, c.After, c.Mode); err != nil {
			return fmt.Errorf("cannot write %s: %w", c.Path, err)
		}
		out.Say("wrote %s", c.Path)
	case KeyCopy:
		return applyCopy(m, out, c)
	case file.UserMkdir:
		if err := m.Mkdir(c.Path, c.Mode); err != nil {
			return fmt.Errorf("cannot make %s: %w", c.Path, err)
		}
	case file.UserChmod:
		if err := m.Chmod(c.Path, c.Mode); err != nil {
			return fmt.Errorf("cannot set %s's mode: %w", c.Path, err)
		}
	case file.UserBackup:
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
