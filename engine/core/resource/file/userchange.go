package file

import (
	"bytes"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// UserWrite puts After in the user's file with Mode. Before is what the file holds, if it exists.
type UserWrite struct {
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

// UserMkdir makes the user's directory.
type UserMkdir struct {
	Path string
	Mode fs.FileMode
}

func (d UserMkdir) String() string { return fmt.Sprintf("would make %s, mode %o", d.Path, d.Mode) }

// Kind is write.
func (UserMkdir) Kind() step.ChangeKind { return step.Write }

// UserChmod sets the mode of the user's file or directory.
type UserChmod struct {
	Path string
	Mode fs.FileMode
}

func (s UserChmod) String() string {
	return fmt.Sprintf("would set %s's mode to %o", s.Path, s.Mode)
}

// Kind is write.
func (UserChmod) Kind() step.ChangeKind { return step.Write }

// UserBackup keeps a copy of the file at To.
type UserBackup struct{ Path, To string }

func (b UserBackup) String() string { return fmt.Sprintf("would keep %s as %s", b.Path, b.To) }
