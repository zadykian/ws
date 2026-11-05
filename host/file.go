package host

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
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

func (writeFile) Kind() step.ChangeKind { return step.Write }

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

// Report compares the file as Plan does. A file that exists and differs stands against ws. A secret
// shows no diff and no sum, as a sum would let a guess be checked.
func (f File) Report(context.Context) ([]step.Item, error) {
	item := step.Item{Kind: step.FileItem, Name: f.Path, Want: attrsText(f.attrs()), Found: missing}
	before, got, ok, err := readRoot(f.M, f.Path)
	if !ok {
		return []step.Item{item}, err
	}
	item.Found = attrsText(got)
	switch {
	case !bytes.Equal(before, f.Content):
		item.Found += ", other content"
		item.Conflict = f.conflict(before)
		if !f.Secret {
			item.Diff = effect.Diff(f.Path, before, f.Content)
		}
	case got != f.attrs():
		item.Conflict = attrsChanged
	}
	return []step.Item{item}, nil
}

// conflict says how the file's content stands against ws. ws keeps no sum of what it wrote, so a
// change by hand reads the same as a file from another version of ws.
func (f File) conflict(before []byte) string {
	if bytes.Contains(f.Content, writtenBy) && !bytes.Contains(before, writtenBy) {
		return "holds no Written by line: ws did not write it"
	}
	return "differs from what ws writes: a change by hand, or another version of ws"
}
