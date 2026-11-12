package host

import (
	"errors"
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

// makeParents makes each directory between home and target that is missing, top first, as the
// owner's at 0755, and leaves one that exists as it is. Without a home it does nothing.
func makeParents(m effect.Machine, home, target string, uid, gid int) error {
	if home == "" {
		return nil
	}
	dirs := chain(home, target)
	for _, at := range dirs[:len(dirs)-1] {
		_, err := m.Stat(at)
		if errors.Is(err, fs.ErrNotExist) {
			err = errors.Join(m.Mkdir(at, 0o755), m.Chown(at, uid, gid), m.Chmod(at, 0o755))
		}
		if err != nil {
			return err
		}
	}
	return nil
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
