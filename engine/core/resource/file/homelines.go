package file

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/internal/paths"
	"github.com/zadykian/ws/engine/core/step"
)

// ErrLinked is a plan's error for a link that lacks ws's lines: a run would replace the link with a
// file. ErrMisplaced is its error for lines that sit where the file's own text wins over them.
var (
	ErrLinked    = errors.New("a link that lacks ws's lines")
	ErrMisplaced = errors.New("ws's lines out of place")
)

// linesError is a plan's stop, with an error kind to match and the words to show.
type linesError struct {
	kind error
	text string
}

func (e linesError) Error() string { return e.text }

func (e linesError) Unwrap() error { return e.kind }

// Misplaced is the error of a Has that finds ws's lines out of place.
func Misplaced(format string, a ...any) error {
	return linesError{ErrMisplaced, fmt.Sprintf(format, a...)}
}

// HomeLines is ws's lines in a file of the account's home that holds other text too. A run adds
// them once, at the top or the end, and keeps the old file, read and written as root as File is. The
// user owns the home, so root refuses a link at the file or on the way, even where the file holds
// the lines, unless Follow is set.
type HomeLines struct {
	M    effect.Machine
	Path string
	// Lines is the text to add. It ends in a newline.
	Lines string
	// Shown is Lines on one line, as the remedy quotes it.
	Shown string
	// Top puts Lines, a blank line and the text. Else the text, a newline if it lacks one, and Lines.
	Top bool
	// Mode is the mode of a new file.
	Mode fs.FileMode
	// Has says whether text holds the lines. It returns Misplaced where they sit out of place.
	Has func(ctx context.Context, m effect.Machine, path string, text []byte) (bool, error)
	// Now stamps the copy of the old file. Nil means time.Now.
	Now func() time.Time
	// UID and GID own a new file. An existing file keeps its owner and mode.
	UID, GID int
	// Home is the account's home, which Path lies below.
	Home string
	// Check, where set, returns the command that refuses a bad file at path, run as root. It runs
	// on the new text at PATH.ws-check before the old file is kept or replaced, so that a refusal
	// leaves the file unchanged.
	Check func(path string) (name string, args []string)
	// Follow reads a link at the file through it, as the running user, where that user owns the
	// home, as on the Mac. A link whose target holds the lines is then in sync, and one that lacks
	// them is refused still. Without it, root reads no file a link of the user's names.
	Follow bool
}

// remedy tells what to do in a file that a run leaves alone.
func (l HomeLines) remedy() string {
	if l.Top {
		return "put '" + l.Shown + "' at the top of that file"
	}
	return "add '" + l.Shown + "' at the end of that file"
}

// after is the text that a run leaves in the file that lacks the lines, as read in found.
func (l HomeLines) after(found Found) []byte {
	if l.Top {
		return append([]byte(l.Lines+"\n"), found.Text...)
	}
	text := append([]byte(nil), found.Text...)
	if len(text) > 0 && text[len(text)-1] != '\n' {
		text = append(text, '\n')
	}
	return append(text, l.Lines...)
}

// keptName is where the old file is copied: PATH.TIME~.
func (l HomeLines) keptName() string {
	now := l.Now
	if now == nil {
		now = time.Now
	}
	return l.Path + "." + now().Format("20060102150405") + "~"
}

// read reads the file as root, through one open below the home that fails at a link, as the user
// may swap one in after the check. A link at the file gives link, and no text unless Follow reads
// through it. A link on the way gives an error. holds says whether the file has the lines. err is
// Misplaced where they sit out of place.
func (l HomeLines) read(ctx context.Context) (found Found, got effect.Attrs, link string, holds bool, err error) {
	info, err := l.M.Stat(l.Path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Found{}, got, "", false, err
	}
	if info.Link != "" {
		return l.readLink(ctx, Found{Exists: true}, info.Link)
	}
	if err = paths.GuardHome(l.M, l.Home, l.Path); err != nil {
		return Found{}, got, "", false, err
	}
	text, got, err := l.M.ReadHomeFile(l.Home, l.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return Found{}, got, "", false, nil
	}
	if err != nil {
		return Found{}, got, "", false, err
	}
	found = Found{Exists: true, Text: text}
	holds, err = l.Has(ctx, l.M, l.Path, text)
	return found, got, "", holds, err
}

// readLink reads the file through a link at it where Follow is set, as the running user. A link
// that leads nowhere lacks the lines.
func (l HomeLines) readLink(ctx context.Context, found Found, link string) (Found, effect.Attrs, string, bool, error) {
	if !l.Follow {
		return found, effect.Attrs{}, link, false, nil
	}
	text, err := l.M.ReadFile(l.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return found, effect.Attrs{}, link, false, nil
	}
	if err != nil {
		return found, effect.Attrs{}, link, false, fmt.Errorf("cannot read %s through its link: %w", l.Path, err)
	}
	found.Text = text
	holds, err := l.Has(ctx, l.M, l.Path, text)
	return found, effect.Attrs{}, link, holds, err
}

// Plan writes the lines where missing: none, a new file, or the old file kept and rewritten.
func (l HomeLines) Plan(ctx context.Context) ([]resource.Change, error) {
	found, got, link, holds, err := l.read(ctx)
	switch {
	case err != nil:
		return nil, err
	case holds:
		return nil, nil
	case !found.Exists:
		// The create comes off this read: a second one, by File.Plan, could follow a link that the
		// user swaps in after the first.
		f := File{M: l.M, Path: l.Path, Content: []byte(l.Lines), Mode: l.Mode, UID: l.UID, GID: l.GID, Home: l.Home}
		if l.Check != nil {
			return []resource.Change{checked{writeFile{f: f, why: "create"}, Validated{File: f, Check: l.Check}}}, nil
		}
		return []resource.Change{writeFile{f: f, why: "create"}}, nil
	case link != "":
		return nil, linesError{ErrLinked, fmt.Sprintf("%s is a link to %s: %s", l.Path, link, l.remedy())}
	}
	after := l.after(found)
	f := File{M: l.M, Path: l.Path, Content: after, Mode: got.Mode, UID: got.UID, GID: got.GID, Home: l.Home}
	keep := homeKeep{M: l.M, Path: l.Path, Home: l.Home, To: l.keptName()}
	if l.Check != nil {
		keep.check = &Validated{File: f, Check: l.Check}
	}
	return []resource.Change{
		keep,
		homeWrite{writeFile{f: f, before: found.Text, why: "write"}},
	}, nil
}

// homeKeep copies the old file to To, with the owner and mode of the read that copies it, as root.
// Where a check is set, it runs first: the copy is the first change of a rewrite.
type homeKeep struct {
	M        effect.Machine
	Path, To string
	Home     string
	check    *Validated
}

func (k homeKeep) String() string { return fmt.Sprintf("would keep %s as %s", k.Path, k.To) }

func (homeKeep) Kind() step.ChangeKind { return step.Write }

// Apply looks for a link on the way again, as the user may make one after Plan. It reads the file
// through one open below the home, so that a link fails, and copies the bytes with the owner and
// mode of that open. A file gone since Plan fails the step.
func (k homeKeep) Apply(ctx context.Context) error {
	if k.check != nil {
		if err := k.check.try(ctx); err != nil {
			return err
		}
	}
	if err := paths.GuardHome(k.M, k.Home, k.Path); err != nil {
		return err
	}
	text, a, err := k.M.ReadHomeFile(k.Home, k.Path)
	if err != nil {
		return err
	}
	return k.M.WriteAtomic(k.To, text, a)
}

// homeWrite is File's write of the file with ws's lines. A log shows ws's lines alone, as the
// rest of the file is the user's and may hold a secret.
type homeWrite struct{ writeFile }

// String is the write's title and what ws adds, as the dry run shows no other line of the file.
func (w homeWrite) String() string { return fmt.Sprintf("%s %s\n%s", w.why, w.f.Path, w.Excerpt()) }

// Title names the file, which the first line of ws's lines in a diff leaves out.
func (w homeWrite) Title() string { return fmt.Sprintf("%s %s", w.why, w.f.Path) }

// Excerpt is the diff of what ws adds to a file it keeps the text of. Such a file is the user's, and
// the lines beside ws's may hold a secret.
func (w homeWrite) Excerpt() string {
	if len(w.before) == 0 || !bytes.Contains(w.f.Content, w.before) {
		return effect.DiffAs(w.f.Path, w.f.Path, w.before, w.f.Content)
	}
	return effect.DiffAs(w.f.Path, w.f.Path, nil, bytes.Replace(w.f.Content, w.before, nil, 1))
}
