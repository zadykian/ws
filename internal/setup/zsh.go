package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

const (
	// zshHeader is the hook's first line, by which a run knows the file for its own.
	zshHeader = "# Written by ws mac apply: its next run replaces changes made here."
	// zshLine is the line of ~/.zshrc that sources the hook.
	zshLine = "source ~/.config/ws/zshrc"
)

// Zsh writes ~/.config/ws/zshrc, a hook that turns off at each prompt the modes that a dropped ssh
// leaves on (docs/mac-ghostty.md). It adds the line that sources the hook at the end of ~/.zshrc,
// and keeps the old file as ~/.zshrc.TIME~, as the ssh step keeps ~/.ssh/config. It warns, and leaves
// ~/.zshrc, where the hook is not ws's or ~/.zshrc is a link that lacks the line.
type Zsh struct {
	Home string
	// Hook is mac/zshrc, its first line zshHeader.
	Hook []byte
	M    effect.Machine
	Out  effect.Out
	// Now stamps the copy of ~/.zshrc. Nil means time.Now.
	Now func() time.Time
}

// Name is zsh.
func (*Zsh) Name() string { return "zsh" }

// WaitsOn is nothing.
func (*Zsh) WaitsOn() string { return "" }

func (z *Zsh) hook() string { return z.Home + "/.config/ws/zshrc" }

func (z *Zsh) rc() string { return z.Home + "/.zshrc" }

// zshOwn says whether the hook's first line is ws's.
func zshOwn(have []byte) bool {
	first, _, _ := strings.Cut(string(have), "\n")
	return first == zshHeader
}

// zshSources says whether a line of text is zshLine, spaces aside.
func zshSources(text []byte) bool {
	for _, line := range strings.Split(string(text), "\n") {
		if strings.TrimSpace(line) == zshLine {
			return true
		}
	}
	return false
}

// Plan plans ~/.config and ~/.config/ws, as the git step does, then the hook and the line.
func (z *Zsh) Plan(context.Context) (step.Plan, error) {
	cs, link, err := wsDirs(z.M, z.Home).plan()
	if err != nil {
		return step.Plan{}, err
	}
	hook, err := z.hookFile().plan()
	switch {
	case errors.Is(err, errNotWs):
		z.Out.Warn("%s is not ws's: move it away, and the next run writes ws's and sources it", z.hook())
		link.warn(z.Out, len(cs), "ws's files go there")
		return step.Plan{Changes: cs}, nil
	case err != nil:
		return step.Plan{}, err
	}
	cs = append(cs, hook...)
	line, err := z.line()
	cs = append(cs, line...)
	link.warn(z.Out, len(cs), "ws's files go there")
	return step.Plan{Changes: cs}, err
}

// line plans zshLine at the end of ~/.zshrc, which it makes where missing.
func (z *Zsh) line() ([]step.Change, error) {
	rc := z.rc()
	info, err := z.M.Stat(rc)
	if errors.Is(err, fs.ErrNotExist) {
		return []step.Change{Write{rc, nil, []byte(zshLine + "\n"), false, 0o644}}, nil
	}
	if err != nil {
		return nil, err
	}
	text, err := z.M.ReadFile(rc)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", rc, err)
	}
	switch {
	case zshSources(text):
		return nil, nil
	case info.Link != "":
		z.Out.Warn("%s is a link to %s: add '%s' at the end of that file", rc, info.Link, zshLine)
		return nil, nil
	}
	now := time.Now
	if z.Now != nil {
		now = z.Now
	}
	after := append([]byte(nil), text...)
	if len(after) > 0 && after[len(after)-1] != '\n' {
		after = append(after, '\n')
	}
	after = append(after, zshLine+"\n"...)
	return []step.Change{Backup{rc, rc + "." + now().Format("20060102150405") + "~"},
		Write{rc, text, after, true, info.Mode.Perm()}}, nil
}

// Apply makes the directories and writes the files.
func (z *Zsh) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		if err := Apply(ctx, z.M, z.Out, c); err != nil {
			return err
		}
	}
	return nil
}

// Report says how the hook and the line stand. A ~/.zshrc that is a link lacking the line stands
// against ws: a run only warns of it.
func (z *Zsh) Report(ctx context.Context) ([]step.Item, error) {
	text, err := z.M.ReadFile(z.rc())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	line := lineItem(z.rc(), zshLine, err == nil, zshSources(text))
	if info, err := z.M.Stat(z.rc()); err == nil && info.Link != "" && line.Found != line.Want {
		line.Conflict = "a link to " + info.Link + ": add the line at the end of that file"
	}
	return gather(ctx, wsDirs(z.M, z.Home), z.hookFile(), fixed{line})
}

// hookFile is the hook, mode 644.
func (z *Zsh) hookFile() userFile {
	return userFile{M: z.M, Path: z.hook(), Text: z.Hook, Mode: 0o644, Own: zshOwn}
}
