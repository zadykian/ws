package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/gitcfg"
)

// Included reports whether text, that of the file name, includes /etc/ws/gitconfig once, at its
// top, as git reads it. Git reads the text on its standard input, so that the answer holds the
// file's bytes alone. Its error names the file and says what is wrong: the include comes twice or
// sits below the file's own settings, or git cannot read the text.
func Included(ctx context.Context, m effect.Machine, name string, text []byte) (bool, error) {
	out, err := m.RunInput(ctx, text, "git", "config", "--file", "-", "--type", "path", "--get-all", "include.path")
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return false, fmt.Errorf("git config cannot read %s: %w", name, err)
	}
	names, err := gitNames(ctx, m, text)
	if err != nil {
		return false, err
	}
	first := ""
	if len(names) > 0 {
		first = names[0]
	}
	holds, err := gitcfg.Included(lines(out), first)
	if err != nil {
		return false, misplaced{fmt.Errorf("%s %w", name, err)}
	}
	return holds, nil
}

// misplaced is Included's error where the include sits out of place, apart from git's failure.
type misplaced struct{ error }

func (e misplaced) Unwrap() error { return e.error }
