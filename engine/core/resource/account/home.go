package account

import (
	"github.com/zadykian/ws/engine/core/resource/file"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/internal/paths"
)

// ReadHome reads the file at rel below the user's home, as root. The user owns the home, and a link
// there would send root's read to whatever it names. So ReadHome refuses a link at the file or on
// the way, and a rel that is not local, before any call to the machine. The read then opens each
// directory below the home, and the file, following no link, so that a link swapped in after that
// check fails too. A missing file gives fs.ErrNotExist.
func (u User) ReadHome(m effect.Machine, rel string) ([]byte, error) {
	at := u.Home + "/" + rel
	if err := paths.GuardHome(m, u.Home, at); err != nil {
		return nil, err
	}
	body, _, err := m.ReadHomeFile(u.Home, at)
	return body, err
}

// File is a file at rel below the account's home, owned by the account. Plan refuses a
// link on the way, as ReadHome does, and a rel that is not local. The caller sets Backup or Secret.
func (a Account) File(m effect.Machine, rel string, content []byte, mode fs.FileMode) file.File {
	return file.File{
		M: m, Path: a.Home + "/" + rel, Content: content, Mode: mode,
		UID: a.UID, GID: a.GID, Home: a.Home,
	}
}

// Dir is a directory at rel below the account's home, whose mode matters: the three at 0700. It
// refuses a link on the way as File does, makes its missing parents at 0755, and sets its own mode
// and owner where they differ.
func (a Account) Dir(m effect.Machine, rel string, mode fs.FileMode) file.Dir {
	return file.Dir{M: m, Path: a.Home + "/" + rel, Mode: mode, UID: a.UID, GID: a.GID, Home: a.Home}
}

// Absent is a path at rel below the account's home that does not exist. It refuses a link on the
// way as File does, as it plans and applies.
func (a Account) Absent(m effect.Machine, rel string) file.Absent {
	return file.Absent{M: m, Path: a.Home + "/" + rel, Home: a.Home}
}

// Lines is ws's lines in the file at rel below the account's home, by l's rules. l gives Lines,
// Shown, Top, Mode, Has and Now. Lines sets the machine, the path, the owner and the home.
// It refuses any link at the file or on the way, as File does.
func (a Account) Lines(m effect.Machine, rel string, l file.UserLines) file.HomeLines {
	l.M, l.Path = m, a.Home+"/"+rel
	return file.HomeLines{Rules: l, UID: a.UID, GID: a.GID, Home: a.Home}
}
