package file

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/internal/paths"
	"github.com/zadykian/ws/engine/core/step"
)

// HomeLines is ws's lines in a file of the account's home that holds other text too, by
// UserLines' rules, read and written as root as File is. The user owns the home, so a link at the
// file or on the way is refused, even where the file holds the lines: root follows none.
type HomeLines struct {
	// Rules gives M, Path, Lines, Shown, Name, Top, Mode (a new file's), Has and Now. It is a field,
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
