// Package file puts files, directories, links and lines on the machine, and removes paths, each
// as a resource that plans its changes and reports its state. The User types do the same in the
// user's home. They read and write as the user: a whole file, a directory, or ws's lines in a file
// that holds other text.
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
	// got is the file's owner and mode, nil for a file that is missing.
	got *effect.Attrs
}

func (c writeFile) String() string {
	if c.f.Secret {
		return fmt.Sprintf("%s %s (content not shown)", c.why, c.f.Path)
	}
	return fmt.Sprintf("%s %s\n%s", c.why, c.f.Path, effect.Diff(c.f.Path, c.before, c.f.Content))
}

func (writeFile) Kind() step.ChangeKind { return step.Write }

// Item is what the File's Report would show: a role's wrapper builds its File as it plans.
func (c writeFile) Item() step.Item { return c.f.item(c.before, c.got) }

// Apply looks for a link on the way again, as the user may make one after Plan. A backup or an edit
// reads as root, outside the bound that the writes keep.
func (c writeFile) Apply(context.Context) error {
	if err := c.f.guard(); err != nil {
		return err
	}
	if err := paths.MakeParents(c.f.M, c.f.Home, c.f.Path, c.f.UID, c.f.GID); err != nil {
		return err
	}
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

// guard refuses a link on the way to a home file. Another File is not checked.
func (f File) guard() error {
	if f.Home == "" {
		return nil
	}
	return paths.GuardHome(f.M, f.Home, f.Path)
}

// Plan reads the file and its attributes as root.
func (f File) Plan(context.Context) ([]resource.Change, error) {
	if err := f.guard(); err != nil {
		return nil, err
	}
	before, err := f.M.ReadRootFile(f.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return []resource.Change{writeFile{f: f, why: "create"}}, nil
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
		return []resource.Change{writeFile{f, before, "write", &got}}, nil
	case got != f.attrs():
		return []resource.Change{writeFile{f, before, fmt.Sprintf("set owner %d:%d and mode %o of", f.UID, f.GID, f.Mode.Perm()), &got}}, nil
	}
	return nil, nil
}

// Report compares the file as Plan does.
func (f File) Report(context.Context) ([]step.Item, error) {
	if err := f.guard(); err != nil {
		return nil, err
	}
	before, got, ok, err := paths.ReadRoot(f.M, f.Path)
	if !ok {
		return []step.Item{f.item(nil, nil)}, err
	}
	return []step.Item{f.item(before, &got)}, nil
}

// item compares what the file holds with what ws writes, got being nil for a file that is missing.
// A file that exists and differs stands against ws. A secret shows no diff and no sum, as a sum
// would let a guess be checked.
func (f File) item(before []byte, got *effect.Attrs) step.Item {
	item := step.Item{Kind: step.FileItem, Name: f.Path, Want: paths.AttrsText(f.attrs()), Found: paths.Missing}
	if got == nil {
		return item
	}
	item.Found = paths.AttrsText(*got)
	switch {
	case !bytes.Equal(before, f.Content):
		item.Found += ", other content"
		item.Conflict = f.conflict(before)
		if !f.Secret {
			item.Diff = effect.Diff(f.Path, before, f.Content)
		}
	case *got != f.attrs():
		item.Conflict = paths.AttrsChanged
	}
	return item
}

// conflict says how the file's content stands against ws. ws keeps no sum of what it wrote, so a
// change by hand reads the same as a file from another version of ws.
func (f File) conflict(before []byte) string {
	switch {
	case f.edit != nil:
		return "a line differs from what ws sets"
	case bytes.Contains(f.Content, writtenBy) && !bytes.Contains(before, writtenBy):
		return "holds no Written by line: ws did not write it"
	}
	return "differs from what ws writes: a change by hand, or another version of ws"
}
