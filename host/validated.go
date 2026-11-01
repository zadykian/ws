package host

import (
	"context"
	"fmt"
	"strings"
)

// Validated is a File that a command checks before it replaces the old file, so that a daemon
// is never left with a configuration it refuses. The command reads the new content from a file
// beside the target, which it removes after.
type Validated struct {
	File
	// Check returns the command, run as root, that exits with an error for a bad file at path.
	Check func(path string) (name string, args []string)
}

type checked struct {
	Change
	v Validated
}

// Apply writes the content beside the target and runs the check on it. Only a good file is then
// written over the target.
func (c checked) Apply(ctx context.Context) error {
	f := c.v.File
	pending := f.Path + ".ws-check"
	if err := f.M.WriteAtomic(pending, f.Content, f.attrs()); err != nil {
		return err
	}
	name, args := c.v.Check(pending)
	out, err := f.M.RunRoot(ctx, name, args...)
	if _, rmErr := f.M.RunRoot(ctx, "rm", "-f", pending); rmErr != nil && err == nil {
		err = rmErr
	}
	if err != nil {
		return fmt.Errorf("%s refuses the new %s: %w: %s", name, f.Path, err, strings.TrimSpace(string(out)))
	}
	return c.Change.Apply(ctx)
}

// Plan plans the file's changes, each checked before it applies.
func (v Validated) Plan(ctx context.Context) ([]Change, error) {
	changes, err := v.File.Plan(ctx)
	for i, c := range changes {
		changes[i] = checked{c, v}
	}
	return changes, err
}
