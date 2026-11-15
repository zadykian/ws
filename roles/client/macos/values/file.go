package values

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/internal/etcws"
)

// ErrRefused is returned where values.yml with the new lines would fail the package's checks, which Out has
// said.
var ErrRefused = errors.New("values.yml would be refused")

// File is values.yml, or a server's file, on a machine, which the Mac's stages add lines to. Stage names the stage that
// holds it, in the messages.
type File struct {
	Path  string
	Stage string
	// Scope is the file that Path is, whose keys the checks take: Both, the zero value, takes any.
	Scope Scope
	M     effect.Machine
	Out   effect.Out
}

// Read returns the text of values.yml, empty where values.yml is missing.
func (f File) Read() (string, error) {
	text, err := f.M.ReadFile(f.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return string(text), err
}

// Compose returns values.yml's text, and its text with the lines appended, where the whole passes the
// package's checks. It appends the lines whose keys values.yml lacks: a line of a key that values.yml
// has stays out, so a value set by hand stays. Where values.yml is a link, as a dotfiles manager makes
// it, and there are lines to add, it says them and stops. f.Stage names the stage in the messages.
func (f File) Compose(lines string) (before, after string, err error) {
	before, err = f.Read()
	if err != nil {
		return "", "", fmt.Errorf("%s: cannot read %s", f.Stage, f.Path)
	}
	var add strings.Builder
	for _, l := range strings.SplitAfter(lines, "\n") {
		if key, ok := lineKey(l); !ok || !HasLine(before, key) {
			add.WriteString(l)
		}
	}
	lines = add.String()
	after = before
	if after != "" && !strings.HasSuffix(after, "\n") {
		after += "\n"
	}
	after += lines
	if _, errs := f.Scope.Parse(after); len(errs) > 0 {
		for _, e := range errs {
			f.Out.Say("%s", e.In(f.Path))
		}
		return "", "", ErrRefused
	}
	if info, err := f.M.Stat(f.Path); err == nil && info.Link != "" && lines != "" {
		f.Out.Say("%s: %s is a link to %s, which the wizard leaves: add these lines to its file, then run the wizard again",
			f.Stage, f.Path, info.Link)
		f.Out.Say("%s", strings.TrimSuffix(lines, "\n"))
		return "", "", fmt.Errorf("%s: %s is a link", f.Stage, f.Path)
	}
	return before, after, nil
}

// Add appends the lines as Compose does. A new file gets mode 644, in a directory it makes.
func (f File) Add(ctx context.Context, lines string) error {
	_, text, err := f.Compose(lines)
	if err != nil {
		return err
	}
	return f.Write(ctx, text)
}

// Write puts the text in values.yml. A file in /etc/ws is root's, 644, written through sudo where it
// differs. Any other is outside /etc/ws, which is the user's: it keeps its mode, as WS_VALUES may
// name it anywhere.
func (f File) Write(ctx context.Context, text string) error {
	if etcws.Holds(f.Path) {
		if err := etcws.Write(ctx, f.M, f.Out, f.Path, []byte(text)); err != nil {
			return fmt.Errorf("cannot write %s: %w", f.Path, err)
		}
		return nil
	}
	mode := fs.FileMode(0o644)
	if info, err := f.M.Stat(f.Path); err == nil {
		mode = info.Mode.Perm()
	}
	if _, err := f.M.Run(ctx, "mkdir", "-p", path.Dir(f.Path)); err != nil {
		return fmt.Errorf("%s: cannot make %s", f.Stage, path.Dir(f.Path))
	}
	if err := f.M.WriteFile(f.Path, []byte(text), mode); err != nil {
		return fmt.Errorf("cannot write %s", f.Path)
	}
	f.Out.Say("wrote %s", f.Path)
	return nil
}

// Load reads values.yml and says each line that the package refuses, which ws client macos apply would
// stop at.
func (f File) Load() (string, Conf, error) {
	text, err := f.Read()
	c, errs := f.Scope.Lines(text)
	for _, e := range errs {
		f.Out.Say("%s", e.In(f.Path))
	}
	if err != nil || len(errs) > 0 {
		return "", c, fmt.Errorf("%s: cannot read %s; make it readable, then run the wizard again", f.Stage, f.Path)
	}
	return text, c, nil
}
