package resource

import (
	"context"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Command runs a program, through sudo where Root is set, as a change that applies itself.
type Command struct {
	M    effect.Machine
	Root bool
	Name string
	Args []string
	// Label is how --check and the run's lines show the command, where it differs from Name and
	// Args. A program's full path, which sudo needs, is long, and a name with spaces is quoted.
	Label string
	// Of is the change's kind. Unset means step.OtherChange.
	Of step.ChangeKind
}

// Line is how --check and a run's lines show the command.
func (c Command) Line() string {
	if c.Label != "" {
		return c.Label
	}
	return strings.Join(append([]string{c.Name}, c.Args...), " ")
}

// String is the line that --check prints.
func (c Command) String() string { return "would run " + c.Line() }

// Kind is Of, or step.OtherChange where Of is unset.
func (c Command) Kind() step.ChangeKind {
	if c.Of == "" {
		return step.OtherChange
	}
	return c.Of
}

// Apply runs the command.
func (c Command) Apply(ctx context.Context) error {
	return c.M.Pass(ctx, c.Root, c.Name, c.Args...)
}
