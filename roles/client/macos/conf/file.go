package conf

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// ErrRefused is returned where mac.conf with the new lines would fail conf's checks, which Out has
// said.
var ErrRefused = errors.New("mac.conf would be refused")

// File is mac.conf on a machine, which the Mac's stages add lines to. Stage names the stage that
// holds it, in the messages.
type File struct {
	Path  string
	Stage string
	M     effect.Machine
	Out   effect.Out
}

// Read returns the text of mac.conf, empty where mac.conf is missing.
func (f File) Read() (string, error) {
	text, err := f.M.ReadFile(f.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return string(text), err
}

// Compose returns mac.conf's text, and its text with the lines appended, whose keys mac.conf lacks,
// where the whole passes conf's checks. Where mac.conf is a link, as a dotfiles manager makes it, and
// there are lines to add, it says them and stops. f.Stage names the stage in the messages.
func (f File) Compose(lines string) (before, after string, err error) {
	before, err = f.Read()
	if err != nil {
		return "", "", fmt.Errorf("%s: cannot read %s", f.Stage, f.Path)
	}
	after = before
	if after != "" && !strings.HasSuffix(after, "\n") {
		after += "\n"
	}
	after += lines
	if _, errs := Parse(after); len(errs) > 0 {
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

// Write puts the text in mac.conf, keeping its mode.
func (f File) Write(ctx context.Context, text string) error {
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

// Load reads mac.conf and says each line that conf refuses, which ws client macos apply would
// stop at.
func (f File) Load() (string, Conf, error) {
	text, err := f.Read()
	c, errs := Lines(text)
	for _, e := range errs {
		f.Out.Say("%s", e.In(f.Path))
	}
	if err != nil || len(errs) > 0 {
		return "", c, fmt.Errorf("%s: cannot read %s; make it readable, then run the wizard again", f.Stage, f.Path)
	}
	return text, c, nil
}
