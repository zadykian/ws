package file

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// UserWrite puts After in the user's file with Mode. Before is what the file holds, if it exists.
type UserWrite struct {
	M             effect.Machine
	Out           effect.Out
	Path          string
	Before, After []byte
	Exists        bool
	Mode          fs.FileMode
}

// String is the diff that --check prints. A file that is new comes from /dev/null.
func (w UserWrite) String() string {
	from := w.Path
	if !w.Exists {
		from = "/dev/null"
	}
	return effect.DiffAs(from, w.Path, w.Before, w.After)
}

// Excerpt is the diff of what ws adds to a file it keeps the text of. Such a file is the user's, and
// the lines beside ws's may hold a secret. ws replaces its own files whole, and their diff stands.
func (w UserWrite) Excerpt() string {
	if len(w.Before) == 0 || !bytes.Contains(w.After, w.Before) {
		return w.String()
	}
	return effect.DiffAs(w.Path, w.Path, nil, bytes.Replace(w.After, w.Before, nil, 1))
}

// Title names the file, which a new file's first line of diff leaves out.
func (w UserWrite) Title() string {
	if !w.Exists {
		return "would create " + w.Path
	}
	return "would write " + w.Path
}

// Kind is write.
func (UserWrite) Kind() step.ChangeKind { return step.Write }

// Apply writes the file, and says so.
func (w UserWrite) Apply(context.Context) error {
	if err := w.M.WriteFile(w.Path, w.After, w.Mode); err != nil {
		return err
	}
	w.Out.Say("wrote %s", w.Path)
	return nil
}

// UserMkdir makes the user's directory.
type UserMkdir struct {
	M    effect.Machine
	Path string
	Mode fs.FileMode
}

func (d UserMkdir) String() string { return fmt.Sprintf("would make %s, mode %o", d.Path, d.Mode) }

// Kind is write.
func (UserMkdir) Kind() step.ChangeKind { return step.Write }

// Apply makes the directory.
func (d UserMkdir) Apply(context.Context) error { return d.M.Mkdir(d.Path, d.Mode) }

// UserChmod sets the mode of the user's file or directory.
type UserChmod struct {
	M    effect.Machine
	Path string
	Mode fs.FileMode
}

func (s UserChmod) String() string {
	return fmt.Sprintf("would set %s's mode to %o", s.Path, s.Mode)
}

// Kind is write.
func (UserChmod) Kind() step.ChangeKind { return step.Write }

// Apply sets the mode.
func (s UserChmod) Apply(context.Context) error { return s.M.Chmod(s.Path, s.Mode) }

// UserBackup keeps a copy of the file at To.
type UserBackup struct {
	M        effect.Machine
	Out      effect.Out
	Path, To string
}

func (b UserBackup) String() string { return fmt.Sprintf("would keep %s as %s", b.Path, b.To) }

// Apply copies the file, with its mode and times, and says so.
func (b UserBackup) Apply(ctx context.Context) error {
	if _, err := b.M.Run(ctx, "cp", "-p", b.Path, b.To); err != nil {
		return err
	}
	b.Out.Say("kept the old %s as %s", b.Path, b.To)
	return nil
}

// applying gives back the user's changes as the resource changes that each of them is, for a plan
// of the user's file, directory or lines.
func applying(cs []step.Change) []resource.Change {
	changes := make([]resource.Change, len(cs))
	for i, c := range cs {
		changes[i] = c.(resource.Change)
	}
	return changes
}
