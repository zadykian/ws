package file

import (
	"bytes"
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
)

// Public is a root's file that everyone may read. Plan reads it, with its owner and mode,
// as the user, so that a run that changes nothing asks for no password. Only a change goes
// through sudo, as File's does.
type Public struct {
	M       effect.Machine
	Path    string
	Content []byte
	Mode    fs.FileMode
}

func (p Public) file() File { return File{M: p.M, Path: p.Path, Content: p.Content, Mode: p.Mode} }

// look reads the file as the user. ok is false where the user may not read it, and got is nil
// where the file is missing.
func (p Public) look() (before []byte, got *effect.Attrs, ok bool, err error) {
	before, err = p.M.ReadFile(p.Path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil, true, nil
	case errors.Is(err, fs.ErrPermission):
		return nil, nil, false, nil
	case err != nil:
		return nil, nil, false, err
	}
	info, err := p.M.Stat(p.Path)
	if err != nil {
		return nil, nil, false, err
	}
	return before, &effect.Attrs{UID: info.UID, GID: info.GID, Mode: info.Mode.Perm()}, true, nil
}

// Plan returns File's change where the file is missing or differs, and none where it holds ws's
// content, owner and mode. A file the user cannot read is planned as root.
func (p Public) Plan(ctx context.Context) ([]resource.Change, error) {
	f := p.file()
	before, got, ok, err := p.look()
	if err != nil {
		return nil, err
	}
	if ok && got != nil && bytes.Equal(before, f.Content) && *got == f.attrs() {
		return nil, nil
	}
	return f.Plan(ctx)
}

// PublicDir is a root's directory that everyone may enter, read as Public's file is.
type PublicDir struct {
	M    effect.Machine
	Path string
	Mode fs.FileMode
}

// Plan returns Dir's change where the directory is missing or differs, and none otherwise.
func (d PublicDir) Plan(ctx context.Context) ([]resource.Change, error) {
	dir := Dir{M: d.M, Path: d.Path, Mode: d.Mode}
	info, err := d.M.Stat(d.Path)
	if err == nil && info.Mode.IsDir() && info.UID == 0 && info.GID == 0 && info.Mode.Perm() == d.Mode.Perm() {
		return nil, nil
	}
	return dir.Plan(ctx)
}
