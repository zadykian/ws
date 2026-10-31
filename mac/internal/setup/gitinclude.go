package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"strings"

	"github.com/zadykian/ws/mac/internal/gitcfg"
)

// gitInclude is what the step read of ~/.gitconfig.
type gitInclude struct {
	Path     string
	Exists   bool
	Mode     fs.FileMode
	Text     []byte
	Included bool
	// Own lists the file's settings of identity and signing.
	Own string
}

// lines runs git config on the file and returns the lines it prints. A status of 1 is no value.
func (g *Git) lines(ctx context.Context, file string, args ...string) ([]string, error) {
	out, err := g.M.Run(ctx, "git", append([]string{"config", "--file", file}, args...)...)
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return nil, fmt.Errorf("git config cannot read %s: %w", file, err)
	}
	var lines []string
	for _, l := range strings.Split(string(out), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines, nil
}

// include reads ~/.gitconfig. It stops where git can't read the file, where the include comes twice
// or sits below the file's own settings, which it then overrides, and where the file is a link that
// lacks the include, rather than replace the link with a file. It writes nothing, so that the step
// stops before it acts, as the ssh step does for ssh's Include.
func (g *Git) include(ctx context.Context) (gitInclude, error) {
	file := g.Home + "/.gitconfig"
	in := gitInclude{Path: file}
	info, err := g.M.Stat(file)
	if missing, err := notExist(err); err != nil || missing {
		return in, err
	}
	in.Exists, in.Mode = true, info.Mode.Perm()
	paths, err := g.lines(ctx, file, "--type", "path", "--get-all", "include.path")
	if err != nil {
		return in, err
	}
	names, err := g.lines(ctx, file, "--name-only", "--list")
	if err != nil {
		return in, err
	}
	first := ""
	if len(names) > 0 {
		first = names[0]
	}
	in.Own = gitcfg.Own(names)
	if in.Included, err = gitcfg.Included(paths, first, g.Home+"/.config/ws"); err != nil {
		return in, fmt.Errorf("%s %w", file, err)
	}
	if !in.Included && info.Link != "" {
		return in, fmt.Errorf("%s is a link to %s: put '[include] path = ~/.config/ws/gitconfig' at the "+
			"top of that file", file, info.Link)
	}
	if in.Text, err = g.M.ReadFile(file); err != nil {
		return in, fmt.Errorf("cannot read %s: %w", file, err)
	}
	return in, nil
}
