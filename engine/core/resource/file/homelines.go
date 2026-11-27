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
// link at the file or on the way, even where the file holds the lines, unless Follow is set.
type HomeLines struct {
	// Rules gives M, Path, Lines, Shown, Name, Top, Mode (a new file's), Has and Now. It stays a field,
	// not embedded, so that UserLines' Read and Changes, which act as the user, stay out.
	Rules UserLines
	// UID and GID own a new file. An existing file keeps its owner and mode.
	UID, GID int
	// Home is the account's home, which Rules.Path lies below.
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

// read reads the file as root, through one open below the home that fails at a link, as the user
// may swap one in after the check. A link at the file gives link, and no text unless Follow reads
// through it. A link on the way gives an error. holds says whether the file has the lines. err is Misplaced where they sit out of
// place.
func (l HomeLines) read(ctx context.Context) (found Found, got effect.Attrs, link string, holds bool, err error) {
	r := l.Rules
	info, err := r.M.Stat(r.Path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Found{}, got, "", false, err
	}
	if info.Link != "" {
		return l.readLink(ctx, Found{Exists: true, Mode: info.Mode}, info.Link)
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

// readLink reads the file through a link at it where Follow is set, as the running user. A link
// that leads nowhere lacks the lines.
func (l HomeLines) readLink(ctx context.Context, found Found, link string) (Found, effect.Attrs, string, bool, error) {
	r := l.Rules
	if !l.Follow {
		return found, effect.Attrs{}, link, false, nil
	}
	text, err := r.M.ReadFile(r.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return found, effect.Attrs{}, link, false, nil
	}
	if err != nil {
		return found, effect.Attrs{}, link, false, fmt.Errorf("cannot read %s through its link: %w", r.Path, err)
	}
	found.Text = text
	holds, err := r.Has(ctx, r.M, r.Path, text)
	return found, effect.Attrs{}, link, holds, err
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
		if l.Check != nil {
			return []resource.Change{checked{writeFile{f: f, why: "create"}, Validated{File: f, Check: l.Check}}}, nil
		}
		return []resource.Change{writeFile{f: f, why: "create"}}, nil
	case link != "":
		return nil, linesError{ErrLinked, fmt.Sprintf("%s is a link to %s: %s", r.Path, link, r.remedy())}
	}
	after := r.After(found, false)
	f := File{M: r.M, Path: r.Path, Content: after, Mode: got.Mode, UID: got.UID, GID: got.GID, Home: l.Home}
	keep := homeKeep{M: r.M, Path: r.Path, Home: l.Home, To: Keep(r.M, nil, r.Path, r.Now).To}
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

// Excerpt is the diff of what ws adds, as UserWrite's.
func (w homeWrite) Excerpt() string {
	return UserWrite{Path: w.f.Path, Before: w.before, After: w.f.Content, Exists: true}.Excerpt()
}
