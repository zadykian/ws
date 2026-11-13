package setup

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
)

// errLinked is a plan's error for a link that lacks ws's lines: a run would replace the link with a
// file. errMisplaced is its error for lines that sit where the file's own text wins over them.
var (
	errLinked    = errors.New("a link that lacks ws's lines")
	errMisplaced = errors.New("ws's lines out of place")
)

// linesError is a plan's stop with the words that a report prints too.
type linesError struct {
	kind error
	text string
}

func (e linesError) Error() string { return e.text }

func (e linesError) Unwrap() error { return e.kind }

// misplaced is the error of a Has that finds ws's lines out of place.
func misplaced(format string, a ...any) error {
	return linesError{errMisplaced, fmt.Sprintf(format, a...)}
}

// userLines is ws's lines in a file of the user's that holds other text too. A run adds them once,
// at the top or the end, and keeps the old file. One read gives the plan and the report.
type userLines struct {
	M    effect.Machine
	Path string
	// Lines is the text to add. It ends in a newline.
	Lines string
	// Shown is Lines on one line, as the remedy quotes it.
	Shown string
	// Name is what the report calls the lines. Empty means Shown.
	Name string
	// Top puts Lines, a blank line and the text. Else the text, a newline if it lacks one, and Lines.
	Top bool
	// Mode is the mode of a new file.
	Mode fs.FileMode
	// Has says whether text holds the lines. It returns misplaced where they sit out of place.
	Has func(ctx context.Context, m effect.Machine, path string, text []byte) (bool, error)
	// Now stamps the copy of the old file. Nil means time.Now.
	Now func() time.Time
}

// remedy tells what to do in a file that a run leaves alone.
func (l userLines) remedy() string {
	if l.Top {
		return "put '" + l.Shown + "' at the top of that file"
	}
	return "add '" + l.Shown + "' at the end of that file"
}

// read reads the file. link is where a link points. holds is whether the file has the lines, and
// err is misplaced where they sit out of place, with found read.
func (l userLines) read(ctx context.Context) (found path, link string, holds bool, err error) {
	info, err := l.M.Stat(l.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return path{}, "", false, nil
	}
	if err != nil {
		return path{}, "", false, err
	}
	found = path{Exists: true, Mode: info.Mode}
	if found.Text, err = l.M.ReadFile(l.Path); err != nil {
		return found, "", false, fmt.Errorf("cannot read %s: %w", l.Path, err)
	}
	holds, err = l.Has(ctx, l.M, l.Path, found.Text)
	return found, info.Link, holds, err
}

// after is the text that a run leaves, as read in found.
func (l userLines) after(found path, holds bool) []byte {
	switch {
	case holds:
		return found.Text
	case !found.Exists:
		return []byte(l.Lines)
	case l.Top:
		return append([]byte(l.Lines+"\n"), found.Text...)
	}
	text := append([]byte(nil), found.Text...)
	if len(text) > 0 && text[len(text)-1] != '\n' {
		text = append(text, '\n')
	}
	return append(text, l.Lines...)
}

// changes plans what read found, or stops at its error, at a link that lacks the lines.
func (l userLines) changes(found path, link string, holds bool, err error) ([]step.Change, error) {
	switch {
	case err != nil:
		return nil, err
	case holds:
		return nil, nil
	case !found.Exists:
		return []step.Change{change.Write{Path: l.Path, Before: nil, After: l.after(found, false), Exists: false, Mode: l.Mode}}, nil
	case link != "":
		return nil, linesError{errLinked, fmt.Sprintf("%s is a link to %s: %s", l.Path, link, l.remedy())}
	}
	return []step.Change{keep(l.Path, l.Now), change.Write{Path: l.Path, Before: found.Text, After: l.after(found, false), Exists: true, Mode: found.Mode.Perm()}}, nil
}

// plan writes the lines where missing: none, a new file, or the old file kept and rewritten.
func (l userLines) plan(ctx context.Context) ([]step.Change, error) {
	found, link, holds, err := l.read(ctx)
	return l.changes(found, link, holds, err)
}

// item says how the file stands against what read found. A conflict is where the plan stops.
func (l userLines) item(found path, link string, holds bool, err error) (step.Item, error) {
	item := lineItem(l.Path, cmp.Or(l.Name, l.Shown), found.Exists, holds)
	switch {
	case errors.Is(err, errMisplaced):
		item.Found, item.Conflict = "out of place", err.Error()
	case err != nil:
		return item, err
	case link != "" && !holds:
		item.Conflict = "a link to " + link + ": " + l.remedy()
	}
	return item, nil
}

// Report reads the file as a plan does.
func (l userLines) Report(ctx context.Context) ([]step.Item, error) {
	found, link, holds, err := l.read(ctx)
	item, err := l.item(found, link, holds, err)
	return []step.Item{item}, err
}

// keep copies the old file to PATH.TIME~. This is the one stamp of the copies.
func keep(file string, now func() time.Time) change.Backup {
	if now == nil {
		now = time.Now
	}
	return change.Backup{Path: file, To: file + "." + now().Format("20060102150405") + "~"}
}
