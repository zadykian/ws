package engine

import (
	"errors"
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Lines prefixes a line with the path of the command that prints it, and says each warning once,
// as a step plans again after it applies.
type Lines struct {
	effect.Out
	// Prog is the prefix: the command's path.
	Prog string
	// Seen holds the warnings said.
	Seen map[string]bool
	// Change says a change under --check in its side's way. It returns the lines, each printed
	// bare, so that an empty one is a blank line. False means a change with no way of its own,
	// which prints as "PATH: change". Nil has none.
	Change func(step.Change) ([]string, bool)
}

// Say prints the line after the prefix.
func (o Lines) Say(format string, a ...any) {
	o.Out.Say("%s: "+format, append([]any{o.Prog}, a...)...)
}

// Observe prints a run's events as the runner's lines once did, but for the line that a step is
// applying. The step's own lines follow that line: a plan under --check, what a step is behind,
// and a left-out step's failed plan.
func (o Lines) Observe(e step.Event) {
	switch {
	case e.Kind == step.Skipped && e.Err != nil:
		o.Warn("left out: %v", e.Err)
	case e.LeftOut:
	case e.Kind == step.Planned && e.Check:
		for _, c := range e.Changes {
			o.sayChange(c)
		}
	case e.Kind == step.Skipped && e.After != "":
		o.Say("%s: after %s", e.Step, e.After)
	}
}

// sayChange prints a change under --check.
func (o Lines) sayChange(c step.Change) {
	if o.Change != nil {
		if lines, ok := o.Change(c); ok {
			for _, line := range lines {
				o.Out.Say("%s", line)
			}
			return
		}
	}
	o.Out.Say("%s: %v", o.Prog, c)
}

// Warn says a warning once.
func (o Lines) Warn(format string, a ...any) {
	line := fmt.Sprintf(format, a...)
	if !o.Seen[line] {
		o.Seen[line] = true
		o.Out.Say("%s: warning: %s", o.Prog, line)
	}
}

// Cause returns the error of a step, without the step and stage that the runner names before it:
// the command names itself.
func Cause(err error) error {
	if inner := errors.Unwrap(err); inner != nil {
		return inner
	}
	return err
}
