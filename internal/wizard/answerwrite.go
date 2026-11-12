package wizard

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/conf"
)

// errRefused is returned where mac.conf with the new lines would fail conf's checks, which Out has
// said.
var errRefused = errors.New("mac.conf would be refused")

// confFile is mac.conf, which the Answers and Mac key stages add lines to.
type confFile struct {
	path  string
	stage string
	m     effect.Machine
	out   effect.Out
}

// read returns the text of mac.conf, empty where mac.conf is missing.
func (f confFile) read() (string, error) {
	text, err := f.m.ReadFile(f.path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return string(text), err
}

// compose returns mac.conf's text, and its text with the lines appended, whose keys mac.conf lacks,
// where the whole passes conf's checks. Where mac.conf is a link, as a dotfiles manager makes it, and
// there are lines to add, it says them and stops. f.stage names the stage in the messages.
func (f confFile) compose(lines string) (before, after string, err error) {
	before, err = f.read()
	if err != nil {
		return "", "", fmt.Errorf("%s: cannot read %s", f.stage, f.path)
	}
	after = before
	if after != "" && !strings.HasSuffix(after, "\n") {
		after += "\n"
	}
	after += lines
	if _, errs := conf.Parse(after); len(errs) > 0 {
		for _, e := range errs {
			f.out.Say("%s", e.In(f.path))
		}
		return "", "", errRefused
	}
	if info, err := f.m.Stat(f.path); err == nil && info.Link != "" && lines != "" {
		f.out.Say("%s: %s is a link to %s, which the wizard leaves: add these lines to its file, then run the wizard again",
			f.stage, f.path, info.Link)
		f.out.Say("%s", strings.TrimSuffix(lines, "\n"))
		return "", "", fmt.Errorf("%s: %s is a link", f.stage, f.path)
	}
	return before, after, nil
}

// add appends the lines as compose does. A new file gets mode 644, in a directory it makes.
func (f confFile) add(ctx context.Context, lines string) error {
	_, text, err := f.compose(lines)
	if err != nil {
		return err
	}
	return f.write(ctx, text)
}

// write puts the text in mac.conf, keeping its mode.
func (f confFile) write(ctx context.Context, text string) error {
	mode := fs.FileMode(0o644)
	if info, err := f.m.Stat(f.path); err == nil {
		mode = info.Mode.Perm()
	}
	if _, err := f.m.Run(ctx, "mkdir", "-p", path.Dir(f.path)); err != nil {
		return fmt.Errorf("%s: cannot make %s", f.stage, path.Dir(f.path))
	}
	if err := f.m.WriteFile(f.path, []byte(text), mode); err != nil {
		return fmt.Errorf("cannot write %s", f.path)
	}
	f.out.Say("wrote %s", f.path)
	return nil
}

// loadConf reads mac.conf and says each line that conf refuses, which ws mac apply would stop at.
func loadConf(f confFile) (string, conf.Conf, error) {
	text, err := f.read()
	c, errs := conf.Lines(text)
	for _, e := range errs {
		f.out.Say("%s", e.In(f.path))
	}
	if err != nil || len(errs) > 0 {
		return "", c, fmt.Errorf("%s: cannot read %s; make it readable, then run the wizard again", f.stage, f.path)
	}
	return text, c, nil
}
