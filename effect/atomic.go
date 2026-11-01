package effect

import (
	"path/filepath"

	"github.com/google/renameio/v2"
)

// WriteAtomic fills a pending file beside path, gives it the owner and mode of a, and renames it
// over path, so that a reader sees the old file or the new one whole. Another owner than ws's own
// needs root.
func (OSMachine) WriteAtomic(path string, data []byte, a Attrs) error {
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
