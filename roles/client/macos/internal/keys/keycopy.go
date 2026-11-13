package keys

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Copy writes ws's copy of the .pub of a key of Secretive's.
type Copy struct {
	M          effect.Machine
	Out        effect.Out
	Path, Text string
}

func (c Copy) String() string {
	return "would write " + c.Path + ", ws's copy of a key of Secretive's"
}

// Kind is write.
func (Copy) Kind() step.ChangeKind { return step.Write }

// Apply writes the copy, and says so.
func (c Copy) Apply(context.Context) error {
	if err := c.M.WriteFile(c.Path, []byte(c.Text), 0o644); err != nil {
		return err
	}
	c.Out.Say("wrote %s", c.Path)
	return nil
}
