package git

import (
	"context"
	"errors"
	"fmt"
	"github.com/zadykian/ws/engine/core/resource/file"
	"os/exec"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/internal/gitcfg"
)

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

// gitconfigLines is the include at the top of ~/.gitconfig. Its place is wrong where it comes twice
// or sits below the file's own settings, which it then overrides. git config reads the file, so the
// step stops before it acts, as the ssh step does for ssh's Include.
func (g *Git) gitconfigLines() file.UserLines {
	return file.UserLines{
		M: g.M, Out: g.Out, Path: g.Home + "/.gitconfig", Lines: gitcfg.Include, Top: true, Mode: 0o644, Now: g.Now,
		Shown: "[include] path = " + gitcfg.Path,
		Name:  "the include of " + gitcfg.Path,
		Has:   g.included,
	}
}

// included says whether the file includes /etc/ws/gitconfig, as git reads it.
func (g *Git) included(ctx context.Context, _ effect.Machine, path string, _ []byte) (bool, error) {
	paths, err := g.lines(ctx, path, "--type", "path", "--get-all", "include.path")
	if err != nil {
		return false, err
	}
	names, err := g.lines(ctx, path, "--name-only", "--list")
	if err != nil {
		return false, err
	}
	first := ""
	if len(names) > 0 {
		first = names[0]
	}
	holds, err := gitcfg.Included(paths, first)
	if err != nil {
		return false, file.Misplaced("%s %v", path, err)
	}
	return holds, nil
}

// includeChanges adds the notes to lines, the changes to ~/.gitconfig, where there are any. It warns
// of the settings of identity and signing in the file itself, which win, and in git's XDG file,
// which lose.
func (g *Git) includeChanges(ctx context.Context, found file.Found, lines []step.Change) []step.Change {
	if len(lines) == 0 {
		return nil
	}
	if rc := g.Home + "/.gitconfig"; found.Exists {
		names, _ := g.lines(ctx, rc, "--name-only", "--list")
		if own := gitcfg.Own(names); own != "" {
			g.note("%s's own settings of %s, below the include, win over %s's",
				rc, own, gitcfg.Path)
		}
	}
	xdg := gitcfg.XDGFile(g.XDG, g.Home)
	if _, err := g.M.Stat(xdg); err == nil {
		names, _ := g.lines(ctx, xdg, "--name-only", "--list")
		if own := gitcfg.Own(names); own != "" {
			g.note("%s", gitcfg.XDGNote(xdg, own, gitcfg.Path))
		}
	}
	return lines
}

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
