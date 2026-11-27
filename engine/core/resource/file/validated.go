package file

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/internal/paths"
	"github.com/zadykian/ws/engine/core/step"
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
	resource.Change
	v Validated
}

// Kind is the wrapped change's: the embedded Change has no Kind to promote.
func (c checked) Kind() step.ChangeKind { return step.KindOf(c.Change) }

// Apply writes the content beside the target and runs the check on it. Only a good file is then
// written over the target.
func (c checked) Apply(ctx context.Context) error {
	if err := c.v.try(ctx); err != nil {
		return err
	}
	return c.Change.Apply(ctx)
}

// try writes the content beside the target, runs the check on it, and removes it. A refusal leaves
// the target unchanged.
func (v Validated) try(ctx context.Context) error {
	f := v.File
	pending := f.Path + ".ws-check"
	if err := f.guard(); err != nil {
		return err
	}
	if err := paths.MakeParents(f.M, f.Home, f.Path, f.UID, f.GID); err != nil {
		return err
	}
	if err := f.M.WriteAtomic(pending, f.Content, f.attrs()); err != nil {
		return err
	}
	name, args := v.Check(pending)
	out, err := f.M.RunRoot(ctx, name, args...)
	if _, rmErr := f.M.RunRoot(ctx, "rm", "-f", pending); rmErr != nil && err == nil {
		err = rmErr
	}
	switch {
	case err != nil && f.Secret:
		// The checker's output may quote the content, which the diff and the log never show. The
		// machine's error holds its stderr, so the exit status alone stands for it.
		return fmt.Errorf("%s refuses the new %s: %s", name, f.Path, exitStatus(err))
	case err != nil:
		return fmt.Errorf("%s refuses the new %s: %w: %s", name, f.Path, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// exitStatus says how a check ended, with none of the text that err carries.
func exitStatus(err error) string {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return fmt.Sprintf("exit status %d", ee.ExitCode())
	}
	return "it failed"
}

// Plan plans the file's changes, each checked before it applies.
func (v Validated) Plan(ctx context.Context) ([]resource.Change, error) {
	changes, err := v.File.Plan(ctx)
	for i, c := range changes {
		changes[i] = checked{c, v}
	}
	return changes, err
}
