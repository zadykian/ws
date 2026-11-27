package kind

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/file"
)

// checkLine refuses a line's fields that no plan could take. The file holds the owner's own text
// too. ws adds its line and leaves the rest, so a field that writes a whole file has no place here.
func (f filePart) checkLine() error {
	switch {
	case f.Line == "":
		return errors.New("has goes with line")
	case f.Template != "" || f.Source != "" || f.Group != "" || f.Secret || f.Absent || f.AbsentWhenEmpty || f.Ours:
		return errors.New("a line takes owner, mode, line_at, check, has, after and when alone")
	case strings.Contains(strings.TrimSuffix(f.Line, "\n"), "\n") && len(f.Has) == 0:
		return errors.New("a line of more than one line takes has, as no line of the file equals it")
	}
	return nil
}

// lines is the file with the line at its top, or at its end with line_at. A missing file is made
// with the line alone, at mode. The old file is kept as PATH.TIME~ before the line goes in. The
// owner may make a link, so a link at the file or on the way is refused. On the Mac, where ws reads
// as the user, a link at the file whose target holds the line is in sync, as a dotfiles manager
// links one.
func (f filePart) lines(e Env, owner account.Account, rel string) file.HomeLines {
	text := f.Line
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	mode := fs.FileMode(f.Mode)
	if mode == 0 {
		mode = 0o644
	}
	l := owner.Lines(e.M, rel, file.UserLines{Lines: text, Shown: shown(text), Top: f.LineAt != "end", Mode: mode, Has: f.holds()})
	l.Check = f.checker()
	l.Follow = e.System == MacOS
	return l
}

// shown is the text on one line, as a report and a remedy quote it.
func shown(text string) string {
	var parts []string
	for l := range strings.Lines(text) {
		if t := strings.TrimSpace(l); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}

// holds says whether a file's text holds the line. Without has, a line of the file holds it where
// it equals the line, spaces aside. With has, the command reads the text on its standard input.
// Exit 0 holds and 1 lacks. Any other fails the plan with the first line of its standard error, so
// that a command can refuse a file whose line sits out of place.
func (f filePart) holds() func(context.Context, effect.Machine, string, []byte) (bool, error) {
	if len(f.Has) == 0 {
		want := strings.TrimSpace(f.Line)
		return func(_ context.Context, _ effect.Machine, _ string, text []byte) (bool, error) {
			for l := range strings.Lines(string(text)) {
				if strings.TrimSpace(l) == want {
					return true, nil
				}
			}
			return false, nil
		}
	}
	return func(ctx context.Context, m effect.Machine, path string, text []byte) (bool, error) {
		_, err := m.RunInput(ctx, text, f.Has[0], f.Has[1:]...)
		var exit *exec.ExitError
		switch {
		case err == nil:
			return true, nil
		case !errors.As(err, &exit):
			return false, fmt.Errorf("has of %s: %w", path, err)
		case exit.ExitCode() == 1:
			return false, nil
		}
		why, _, _ := strings.Cut(strings.TrimSpace(string(exit.Stderr)), "\n")
		return false, fmt.Errorf("%s refuses %s, exit status %d: %s", f.Has[0], path, exit.ExitCode(), why)
	}
}
