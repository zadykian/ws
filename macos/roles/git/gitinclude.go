package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/gitcfg"
)

// Included says whether the git config text includes /etc/ws/gitconfig, as git reads it from
// stdin. An include below the text's own settings, or one that comes twice, is an error, as is a
// text that git cannot read.
func Included(ctx context.Context, m effect.Machine, text []byte) (bool, error) {
	var got [2][]string
	for i, args := range [][]string{{"--type", "path", "--get-all", "include.path"}, {"--name-only", "--list"}} {
		out, err := m.RunInput(ctx, text, "git", append([]string{"config", "--file", "-"}, args...)...)
		var ee *exec.ExitError
		if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
			return false, fmt.Errorf("git config cannot read ~/.gitconfig: %w", err)
		}
		got[i] = strings.Fields(string(out))
	}
	first := ""
	if len(got[1]) > 0 {
		first = got[1][0]
	}
	holds, err := gitcfg.Included(got[0], first)
	if err != nil {
		return false, fmt.Errorf("~/.gitconfig %w", err)
	}
	return holds, nil
}
