package host

import (
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host/internal/paths"
)

// ReadHome reads the file at rel below the user's home, as root. The user owns the home, and a link
// there would send root's read to whatever it names. So ReadHome refuses a link at the file or on
// the way, and a rel that is not local, before any call to the machine. A missing file gives
// fs.ErrNotExist.
func (u User) ReadHome(m effect.Machine, rel string) ([]byte, error) {
	at := u.Home + "/" + rel
	if err := paths.GuardHome(m, u.Home, at); err != nil {
		return nil, err
	}
	return m.ReadRootFile(at)
}

// File is a file at rel below the account's home, owned by the account. Plan and Report refuse a
// link on the way, as ReadHome does, and a rel that is not local. The caller sets Backup or Secret.
func (a Account) File(m effect.Machine, rel string, content []byte, mode fs.FileMode) File {
	return File{
		M: m, Path: a.Home + "/" + rel, Content: content, Mode: mode,
		UID: a.UID, GID: a.GID, Home: a.Home,
	}
}

// Dir is a directory at rel below the account's home, whose mode matters: the three at 0700. It
// refuses a link on the way as File does, makes its missing parents at 0755, and sets its own mode
// and owner where they differ.
func (a Account) Dir(m effect.Machine, rel string, mode fs.FileMode) Dir {
	return Dir{M: m, Path: a.Home + "/" + rel, Mode: mode, UID: a.UID, GID: a.GID, Home: a.Home}
}

// Absent is a path at rel below the account's home that does not exist. It refuses a link on the
// way as File does, as it plans, reports and applies.
func (a Account) Absent(m effect.Machine, rel string) Absent {
	return Absent{M: m, Path: a.Home + "/" + rel, Home: a.Home}
}
