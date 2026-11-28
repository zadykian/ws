// Package file puts files, directories, links and lines on the machine, and removes paths, each
// as a resource that plans its changes. HomeLines adds ws's lines to a file of the account's home.
package file

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/internal/paths"
	"github.com/zadykian/ws/engine/core/step"
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
	// Create makes the file through WriteNew, so that Apply fails where anything stands there by
	// then, and Plan fails where it stands now. It leaves Backup and edit unused.
	Create bool
	// edit, where set, makes the content from the file as Apply finds it, so that two resources
	// of one file, planned together, keep each other's edit.
	edit func(before []byte) []byte
	// Home, set by Account.File alone, is the home that the path lies below.
	Home string
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

func (writeFile) Kind() step.ChangeKind { return step.Write }

// Apply looks for a link on the way again, as the user may make one after Plan. A backup or an edit
// reads as root, a home file through one open below the home, as the writes keep their bound.
func (c writeFile) Apply(context.Context) error {
	if err := c.f.guard(); err != nil {
		return err
	}
	if err := paths.MakeParents(c.f.M, c.f.Home, c.f.Path, c.f.UID, c.f.GID); err != nil {
		return err
	}
	if c.f.edit != nil {
		before, _, err := c.f.read()
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
	if c.f.Create {
		return c.f.M.WriteNew(c.f.Path, c.f.Content, c.f.attrs())
	}
	return c.f.M.WriteAtomic(c.f.Path, c.f.Content, c.f.attrs())
}

// guard refuses a link on the way to a home file. Another File is not checked.
func (f File) guard() error {
	if f.Home == "" {
		return nil
	}
	return paths.GuardHome(f.M, f.Home, f.Path)
}

// read reads the file with its owner and mode as root. A home file goes through one open below the
// home, as the account may swap a link in after the guard. Another file is read by name.
func (f File) read() ([]byte, effect.Attrs, error) {
	if f.Home != "" {
		return f.M.ReadHomeFile(f.Home, f.Path)
	}
	before, err := f.M.ReadRootFile(f.Path)
	if err != nil {
		return nil, effect.Attrs{}, err
	}
	got, err := f.M.RootAttrs(f.Path)
	return before, got, err
}

// Plan reads the file and its attributes as root.
func (f File) Plan(context.Context) ([]resource.Change, error) {
	if err := f.guard(); err != nil {
		return nil, err
	}
	before, got, err := f.read()
	if errors.Is(err, fs.ErrNotExist) {
		return []resource.Change{writeFile{f: f, why: "create"}}, nil
	}
	if err != nil {
		return nil, err
	}
	if f.Create {
		return nil, fmt.Errorf("%s exists already: %w", f.Path, fs.ErrExist)
	}
	switch {
	case !bytes.Equal(before, f.Content):
		return []resource.Change{writeFile{f, before, "write"}}, nil
	case got != f.attrs():
		return []resource.Change{writeFile{f, before, fmt.Sprintf("set owner %d:%d and mode %o of", f.UID, f.GID, f.Mode.Perm())}}, nil
	}
	return nil, nil
}
