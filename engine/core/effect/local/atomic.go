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
	if root, rel, err := m.confined(path); err != nil || root != nil {
		if err != nil {
			return err
		}
		defer func() { _ = root.Close() }()
		return writeAtomicIn(root, rel, data, a)
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

// Chown sets the owner of the file a link points to. It needs root for another owner than ws's,
// which sudo gives where ws runs as a user.
func (m OSMachine) Chown(path string, uid, gid int) error {
	if m.asRoot(path) && another(path, uid) {
		_, err := m.RunRoot(context.Background(), "chown", "--", fmt.Sprintf("%d:%d", uid, gid), path)
		return err
	}
	root, rel, err := m.confined(path)
	if err != nil {
		return err
	}
	if root == nil {
		return os.Chown(path, uid, gid)
	}
	defer func() { _ = root.Close() }()
	return root.Chown(rel, uid, gid)
}

// Chmod sets the mode of the file a link points to, through sudo where ws runs as a user and the
// file is another's.
func (m OSMachine) Chmod(path string, perm fs.FileMode) error {
	if m.asRoot(path) && another(path, -1) {
		_, err := m.RunRoot(context.Background(), "chmod", "--", strconv.FormatUint(uint64(perm.Perm()), 8), path)
		return err
	}
	root, rel, err := m.confined(path)
	if err != nil {
		return err
	}
	if root == nil {
		return os.Chmod(path, perm)
	}
	defer func() { _ = root.Close() }()
	return root.Chmod(rel, perm)
}

// Symlink makes path a link to target, replacing a file or link there through a new link renamed
// over it.
func (m OSMachine) Symlink(target, path string) error {
	if root, rel, err := m.confined(path); err != nil || root != nil {
		if err != nil {
			return err
		}
		defer func() { _ = root.Close() }()
		tmp := filepath.Join(filepath.Dir(rel), "."+filepath.Base(rel)+".ws-new")
		_ = root.Remove(tmp)
		if err := root.Symlink(target, tmp); err != nil {
			return err
		}
		if err := root.Rename(tmp, rel); err != nil {
			_ = root.Remove(tmp)
			return err
		}
		return nil
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
