package host

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// ReadHome reads the file at rel below the user's home, as root. The user owns the home, and a link
// there would send root's read to whatever it names. So ReadHome refuses a link at the file or on
// the way, and a rel that is not local, before any call to the machine. A missing file gives
// fs.ErrNotExist.
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

// Dir is a directory at rel below the account's home, whose mode matters: the three at 0700. It
// refuses a link on the way as File does, makes its missing parents at 0755, and sets its own mode
// and owner where they differ.
func (a Account) Dir(m effect.Machine, rel string, mode fs.FileMode) Dir {
	return Dir{M: m, Path: a.Home + "/" + rel, Mode: mode, UID: a.UID, GID: a.GID, home: a.Home}
}

// Absent is a path at rel below the account's home that does not exist. It refuses a link on the
// way as File does, as it plans, reports and applies.
func (a Account) Absent(m effect.Machine, rel string) Absent {
	return Absent{M: m, Path: a.Home + "/" + rel, home: a.Home}
}

// makeParents makes each directory between home and target that is missing, top first, as the
// owner's at 0755, and leaves an existing one alone. Without a home it does nothing.
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
	for _, at := range chain(home, target) {
		info, err := m.Stat(at)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("look for a link at %s: %w", at, err)
		}
		if info.Link != "" {
			return fmt.Errorf("%s is a link to %s: replace it with the file itself", at, info.Link)
		}
	}
	return nil
}

// chain returns each directory below home on the way to target, then target.
func chain(home, target string) []string {
	rel, ok := strings.CutPrefix(target, home+"/")
	if !ok {
		return []string{target}
	}
	var list []string
	at := home
	for part := range strings.SplitSeq(rel, "/") {
		at = path.Join(at, part)
		list = append(list, at)
	}
	return list
}
