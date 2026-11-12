package host

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/zadykian/ws/effect"
)

// ReadHome reads the file at rel below the user's home, as root. The user owns the home, so a link
// at the file or at a directory on the way would send root's read to whatever it names: ReadHome
// refuses one, and a rel that is not local, before any call to the machine. A file that is missing
// gives fs.ErrNotExist.
func (u User) ReadHome(m effect.Machine, rel string) ([]byte, error) {
	at := u.Home + "/" + rel
	if err := guardHome(m, u.Home, at); err != nil {
		return nil, err
	}
	return m.ReadRootFile(at)
}

// File is a file at rel below the account's home, owned by the account. Plan and Report refuse a
// link on the way, as ReadHome does, and a rel that is not local. The caller sets Backup or Secret.
func (a Account) File(m effect.Machine, rel string, content []byte, mode fs.FileMode) File {
	return File{
		M: m, Path: a.Home + "/" + rel, Content: content, Mode: mode,
		UID: a.UID, GID: a.GID, home: a.Home,
	}
}

// guardHome fails unless target lies strictly below home by a local path, and fails on the first
// link at target or at a directory between home and it.
func guardHome(m effect.Machine, home, target string) error {
	rel, ok := strings.CutPrefix(target, home+"/")
	if !ok || !filepath.IsLocal(rel) {
		return fmt.Errorf("%s is not a path below the home %s", target, home)
	}
	return RefuseLinks(m, home, target)
}
