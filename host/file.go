package host

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/effect"
)

// File is a file with its content, mode and owner.
type File struct {
	M       effect.Machine
	Path    string
	Content []byte
	Mode    fs.FileMode
	// UID and GID are the owner, root's by default.
	UID, GID int
	// Secret keeps the content out of --check's diff, for a password or a key.
	Secret bool
	// Backup keeps the old file beside the new one, as PATH.PID.TIME~, where the content changes.
	Backup bool
	// edit, where set, makes the content from the file as Apply finds it, so that two resources
	// of one file, planned together, keep each other's edit.
	edit func(before []byte) []byte
}

func (f File) attrs() effect.Attrs { return effect.Attrs{UID: f.UID, GID: f.GID, Mode: f.Mode.Perm()} }

type writeFile struct {
	f      File
	before []byte
	why    string
}

func (c writeFile) String() string {
	if c.f.Secret {
		return fmt.Sprintf("%s %s (content not shown)", c.why, c.f.Path)
	}
	return fmt.Sprintf("%s %s\n%s", c.why, c.f.Path, effect.Diff(c.f.Path, c.before, c.f.Content))
}

func (c writeFile) Apply(context.Context) error {
	if c.f.edit != nil {
		before, err := c.f.M.ReadRootFile(c.f.Path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		c.f.Content = c.f.edit(before)
	}
	if c.f.Backup {
		if err := c.f.backup(c.f.Content); err != nil {
			return err
		}
	}
	return c.f.M.WriteAtomic(c.f.Path, c.f.Content, c.f.attrs())
}

// Plan reads the file and its attributes as root.
func (f File) Plan(context.Context) ([]Change, error) {
	before, err := f.M.ReadRootFile(f.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return []Change{writeFile{f: f, why: "create"}}, nil
	}
	if err != nil {
		return nil, err
	}
	got, err := f.M.RootAttrs(f.Path)
	if err != nil {
		return nil, err
	}
	switch {
	case !bytes.Equal(before, f.Content):
		return []Change{writeFile{f, before, "write"}}, nil
	case got != f.attrs():
		return []Change{writeFile{f, before, fmt.Sprintf("set owner %d:%d and mode %o of", f.UID, f.GID, f.Mode.Perm())}}, nil
	}
	return nil, nil
}
