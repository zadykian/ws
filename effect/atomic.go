package effect

import (
	"os"
	"path/filepath"

	"github.com/google/renameio/v2"
)

// WriteAtomic fills a pending file beside path, gives it the owner and mode of a, and renames it
// over path, so that a reader sees the old file or the new one whole. Another owner than ws's own
// needs root.
func (m OSMachine) WriteAtomic(path string, data []byte, a Attrs) error {
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

// Chown sets the owner of the file a link points to. It needs root for another owner than ws's.
func (OSMachine) Chown(path string, uid, gid int) error { return os.Chown(path, uid, gid) }

// Symlink makes path a link to target, replacing a file or link there through a new link renamed
// over it.
func (m OSMachine) Symlink(target, path string) error {
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
