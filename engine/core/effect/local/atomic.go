// Package local is the effect.Machine of the machine that ws runs on: files, commands, sudo and
// downloads.
package local

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/renameio/v2"

	"github.com/zadykian/ws/engine/core/effect"
)

// WriteAtomic fills a pending file beside path, gives it the owner and mode of a, and renames it
// over path. So a reader sees the old file or the new one whole. Another owner than ws's own
// needs root, which sudo gives where ws runs as a user.
func (m OSMachine) WriteAtomic(path string, data []byte, a effect.Attrs) error {
	if a.UID >= 0 && m.asRoot(path) && another(path, a.UID) {
		return m.sudoWrite(path, data, fmt.Sprintf("%d:%d", a.UID, a.GID), a.Mode)
	}
	f, err := renameio.NewPendingFile(path,
		renameio.WithTempDir(filepath.Dir(path)), renameio.WithPermissions(a.Mode.Perm()))
	if err != nil {
		return err
	}
	defer func() { _ = f.Cleanup() }()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Chmod(a.Mode.Perm()); err != nil {
		return err
	}
	if err := f.Chown(a.UID, a.GID); err != nil {
		return err
	}
	return f.CloseAtomicallyReplace()
}

// WriteNew fills a pending file beside path, as WriteAtomic does, and links it to path, which
// fails where anything stands there. So two writers of one path never replace each other's file.
// It writes as ws's own user, and refuses another owner, which only sudo could give.
func (OSMachine) WriteNew(path string, data []byte, a effect.Attrs) error {
	if a.UID >= 0 && a.UID != euid() {
		return fmt.Errorf("write %s: only ws's own owner, %d, not %d", path, euid(), a.UID)
	}
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".new-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Chmod(a.Mode.Perm()); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Chown(a.UID, a.GID); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Link(f.Name(), path)
}

// Chown sets the owner of the file a link points to. It needs root for another owner than ws's,
// which sudo gives where ws runs as a user.
func (m OSMachine) Chown(path string, uid, gid int) error {
	if m.asRoot(path) && another(path, uid) {
		_, err := m.RunRoot(context.Background(), "chown", "--", fmt.Sprintf("%d:%d", uid, gid), path)
		return err
	}
	return os.Chown(path, uid, gid)
}

// Chmod sets the mode of the file a link points to, through sudo where ws runs as a user and the
// file is another's.
func (m OSMachine) Chmod(path string, perm fs.FileMode) error {
	if m.asRoot(path) && another(path, -1) {
		_, err := m.RunRoot(context.Background(), "chmod", "--", strconv.FormatUint(uint64(perm.Perm()), 8), path)
		return err
	}
	return os.Chmod(path, perm)
}

// linkScript is POSIX sh, run as root with the target and the path: it refuses a directory at the
// path that is no link, else links.
const linkScript = `if [ -d "$2" ] && [ ! -L "$2" ]; then echo "$2 is a directory, not a link" >&2; exit 1; fi
exec ln -sfn -- "$1" "$2"`

// Symlink makes path a link to target, replacing a file or link there through a new link renamed
// over it. Where ws runs as a user and may not write path's directory, sudo's ln does it, which
// replaces the link in two steps.
func (m OSMachine) Symlink(target, path string) error {
	if m.asRoot(path) && unwritable(filepath.Dir(path)) {
		// ln -n puts the link inside a real directory, and exits 0 as if it had replaced it. Root
		// checks, as the user may not search a directory above path.
		_, err := m.RunRoot(context.Background(), "sh", "-c", linkScript, "sh", target, path)
		return err
	}
	tmp := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".ws-new")
	_ = os.Remove(tmp)
	if err := os.Symlink(target, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
