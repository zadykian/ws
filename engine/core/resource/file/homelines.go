package file

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/internal/paths"
	"github.com/zadykian/ws/engine/core/step"
)

// HomeLines is ws's lines in a file of the account's home that holds other text too, by
// UserLines' rules, read and written as root as File is. The user owns the home, so root refuses a
// link at the file or on the way, even where the file holds the lines.
type HomeLines struct {
	// Rules gives M, Path, Lines, Shown, Name, Top, Mode (a new file's), Has and Now. It stays a field,
	// not embedded, so that UserLines' Read and Changes, which act as the user, stay out.
	Rules UserLines
	// UID and GID own a new file. An existing file keeps its owner and mode.
	UID, GID int
	// Home is the account's home, which Rules.Path lies below.
	Home string
}

// read reads the file as root, through one open below the home that fails at a link, as the user
// may swap one in after the check. A link at the file gives link and no text, and a link on the way
// gives an error. holds says whether the file has the lines. err is Misplaced where they sit out of
// place.
func (l HomeLines) read(ctx context.Context) (found Found, got effect.Attrs, link string, holds bool, err error) {
	r := l.Rules
	info, err := r.M.Stat(r.Path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Found{}, got, "", false, err
	}
	if info.Link != "" {
		return Found{Exists: true, Mode: info.Mode}, got, info.Link, false, nil
	}
	if err = paths.GuardHome(r.M, l.Home, r.Path); err != nil {
		return Found{}, got, "", false, err
	}
	text, got, err := r.M.ReadHomeFile(l.Home, r.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return Found{}, got, "", false, nil
	}
	if err != nil {
		return Found{}, got, "", false, err
	}
	found = Found{Exists: true, Mode: got.Mode, Text: text}
	holds, err = r.Has(ctx, r.M, r.Path, text)
	return found, got, "", holds, err
}

// Report reads the file as a plan does.
func (l HomeLines) Report(ctx context.Context) ([]step.Item, error) {
	found, _, link, holds, err := l.read(ctx)
	item, err := l.Rules.Item(found, link, holds, err)
	return []step.Item{item}, err
}

// Plan writes the lines where missing: none, a new file, or the old file kept and rewritten.
func (l HomeLines) Plan(ctx context.Context) ([]resource.Change, error) {
	found, got, link, holds, err := l.read(ctx)
	r := l.Rules
	switch {
	case err != nil:
		return nil, err
	case holds:
		return nil, nil
	case !found.Exists:
		// The create comes off this read: a second one, by File.Plan, could follow a link that the
		// user swaps in after the first.
		f := File{M: r.M, Path: r.Path, Content: []byte(r.Lines), Mode: r.Mode, UID: l.UID, GID: l.GID, Home: l.Home}
		return []resource.Change{writeFile{f: f, why: "create"}}, nil
	case link != "":
		return nil, linesError{ErrLinked, fmt.Sprintf("%s is a link to %s: %s", r.Path, link, r.remedy())}
	}
	after := r.After(found, false)
	f := File{M: r.M, Path: r.Path, Content: after, Mode: got.Mode, UID: got.UID, GID: got.GID, Home: l.Home}
	return []resource.Change{
		homeKeep{M: r.M, Path: r.Path, Home: l.Home, To: Keep(r.M, nil, r.Path, r.Now).To},
		homeWrite{writeFile{f: f, before: found.Text, why: "write", got: &got}},
	}, nil
}

// homeKeep copies the old file to To, with the owner and mode of the read that copies it, as root.
type homeKeep struct {
	M        effect.Machine
	Path, To string
	Home     string
}

func (k homeKeep) String() string { return fmt.Sprintf("would keep %s as %s", k.Path, k.To) }

func (homeKeep) Kind() step.ChangeKind { return step.Write }

// Apply looks for a link on the way again, as the user may make one after Plan. It reads the file
// through one open below the home, so that a link fails, and copies the bytes with the owner and
// mode of that open. A file gone since Plan fails the step.
func (k homeKeep) Apply(context.Context) error {
	if err := paths.GuardHome(k.M, k.Home, k.Path); err != nil {
		return err
	}
	text, a, err := k.M.ReadHomeFile(k.Home, k.Path)
	if err != nil {
		return err
	}
	return k.M.WriteAtomic(k.To, text, a)
}

// homeWrite is File's write of the file with ws's lines. A report shows ws's lines alone, as the
// rest of the file is the user's and may hold a secret.
type homeWrite struct{ writeFile }

// String is the write's title and what ws adds, as the dry run shows no other line of the file.
func (w homeWrite) String() string { return fmt.Sprintf("%s %s\n%s", w.why, w.f.Path, w.Excerpt()) }

// Title names the file, which the first line of ws's lines in a diff leaves out.
func (w homeWrite) Title() string { return fmt.Sprintf("%s %s", w.why, w.f.Path) }

// Item is the file's item without its diff, as the state report and its JSON hold no other line of
// the file.
func (w homeWrite) Item() step.Item {
	item := w.writeFile.Item()
	item.Diff = ""
	return item
}

// Excerpt is the diff of what ws adds, as UserWrite's.
func (w homeWrite) Excerpt() string {
	return UserWrite{Path: w.f.Path, Before: w.before, After: w.f.Content, Exists: true}.Excerpt()
}
