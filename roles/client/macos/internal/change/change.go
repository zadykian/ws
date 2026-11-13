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
	"github.com/zadykian/ws/roles/client/macos/internal/keys"
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
		if err := c.Apply(ctx); err != nil {
			return fmt.Errorf("cannot write %s: %w", c.Path, err)
		}
	case keys.Copy:
		if err := c.Apply(ctx); err != nil {
			return fmt.Errorf("cannot write %s: %w", c.Path, err)
		}
	case file.UserMkdir:
		if err := c.Apply(ctx); err != nil {
			return fmt.Errorf("cannot make %s: %w", c.Path, err)
		}
	case file.UserChmod:
		if err := c.Apply(ctx); err != nil {
			return fmt.Errorf("cannot set %s's mode: %w", c.Path, err)
		}
	case file.UserBackup:
		if err := c.Apply(ctx); err != nil {
			return fmt.Errorf("cannot keep %s as %s: %w", c.Path, c.To, err)
		}
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
