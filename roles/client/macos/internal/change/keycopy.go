package change

import (
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
)

// KeyCopy writes ws's copy of the .pub of a key of Secretive's.
type KeyCopy struct{ Path, Text string }

func (c KeyCopy) String() string {
	return "would write " + c.Path + ", ws's copy of a key of Secretive's"
}

func applyCopy(m effect.Machine, out effect.Out, c KeyCopy) error {
	if err := m.WriteFile(c.Path, []byte(c.Text), 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", c.Path, err)
	}
	out.Say("wrote %s", c.Path)
	return nil
}
