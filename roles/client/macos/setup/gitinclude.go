package setup

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/gitcfg"
)

// includeText is what goes at the top of ~/.gitconfig, so that the file's own settings come after
// the include and win.
const includeText = "[include]\n    path = ~/.config/ws/gitconfig\n"

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
func (g *Git) gitconfigLines() userLines {
	return userLines{
		M: g.M, Path: g.Home + "/.gitconfig", Lines: includeText, Top: true, Mode: 0o644, Now: g.Now,
		Shown: "[include] path = ~/.config/ws/gitconfig",
		Name:  "the include of ~/.config/ws/gitconfig",
		Has:   g.included,
	}
}

// included says whether the file includes ~/.config/ws/gitconfig, as git reads it.
func (g *Git) included(ctx context.Context, _ effect.Machine, file string, _ []byte) (bool, error) {
	paths, err := g.lines(ctx, file, "--type", "path", "--get-all", "include.path")
	if err != nil {
		return false, err
	}
	names, err := g.lines(ctx, file, "--name-only", "--list")
	if err != nil {
		return false, err
	}
	first := ""
	if len(names) > 0 {
		first = names[0]
	}
	holds, err := gitcfg.Included(paths, first, g.Home+"/.config/ws")
	if err != nil {
		return false, misplaced("%s %v", file, err)
	}
	return holds, nil
}

// includeChanges adds the notes to lines, the changes to ~/.gitconfig, where there are any. It warns
// of the settings of identity and signing in the file itself, which win, and in git's XDG file,
// which lose.
func (g *Git) includeChanges(ctx context.Context, found path, lines []step.Change) []step.Change {
	if len(lines) == 0 {
		return nil
	}
	if file := g.Home + "/.gitconfig"; found.Exists {
		names, _ := g.lines(ctx, file, "--name-only", "--list")
		if own := gitcfg.Own(names); own != "" {
			g.note("%s's own settings of %s, below the include, win over ~/.config/ws/gitconfig's",
				file, own)
		}
	}
	xdg := g.XDG
	if xdg == "" {
		xdg = g.Home + "/.config"
	}
	if _, err := g.M.Stat(xdg + "/git/config"); err == nil {
		names, _ := g.lines(ctx, xdg+"/git/config", "--name-only", "--list")
		if own := gitcfg.Own(names); own != "" {
			g.note("%s/git/config's settings of %s come before the include in ~/.gitconfig, so "+
				"~/.config/ws/gitconfig's win over them", xdg, own)
		}
	}
	return lines
}
