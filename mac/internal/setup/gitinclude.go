package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"strings"
	"time"

	"github.com/zadykian/ws/mac/internal/gitcfg"
	"github.com/zadykian/ws/mac/internal/step"
)

// includeText is what goes at the top of ~/.gitconfig, so that the file's own settings come after
// the include and win.
const includeText = "[include]\n    path = ~/.config/ws/gitconfig\n"

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

// includeChanges puts an include of ~/.config/ws/gitconfig at the top of ~/.gitconfig where there is
// none. It makes the file where missing, and keeps the old one as ~/.gitconfig.TIME~, as the ssh step
// keeps ~/.ssh/config. It warns of the settings of identity and signing in the file itself, which
// win, and in git's XDG file, which lose.
func (g *Git) includeChanges(ctx context.Context, in gitInclude) []step.Change {
	if in.Included {
		return nil
	}
	now := time.Now
	if g.Now != nil {
		now = g.Now
	}
	mode := fs.FileMode(0o644)
	var cs []step.Change
	text := includeText
	if in.Exists {
		mode = in.Mode
		cs = append(cs, Backup{in.Path, in.Path + "." + now().Format("20060102150405") + "~"})
		text += "\n" + string(in.Text)
	}
	cs = append(cs, Write{in.Path, in.Text, []byte(text), in.Exists, mode})
	if in.Own != "" {
		g.note("%s's own settings of %s, below the include, win over ~/.config/ws/gitconfig's",
			in.Path, in.Own)
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
	return cs
}
