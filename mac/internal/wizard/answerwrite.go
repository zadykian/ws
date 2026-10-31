package wizard

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
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

// read returns the text of mac.conf, empty where it is missing.
func (f confFile) read() (string, error) {
	text, err := f.m.ReadFile(f.path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return string(text), err
}

// add appends the lines, whose keys mac.conf lacks, where the whole passes conf's checks. A new file
// gets mode 644, in a directory it makes. Where mac.conf is a link, as a dotfiles manager makes it,
// it says the lines and stops. f.stage names the stage in the messages.
func (f confFile) add(ctx context.Context, lines string) error {
	text, err := f.read()
	if err != nil {
		return fmt.Errorf("%s: cannot read %s", f.stage, f.path)
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += lines
	if _, errs := conf.Parse(text); len(errs) > 0 {
		for _, e := range errs {
			f.out.Say("%s", e.In(f.path))
		}
		return errRefused
	}
	info, err := f.m.Stat(f.path)
	if err == nil && info.Link != "" {
		f.out.Say("%s: %s is a link to %s, which the wizard leaves: add these lines to its file, then run the wizard again",
			f.stage, f.path, info.Link)
		f.out.Say("%s", strings.TrimSuffix(lines, "\n"))
		return fmt.Errorf("%s: %s is a link", f.stage, f.path)
	}
	mode := fs.FileMode(0o644)
	if err == nil {
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
