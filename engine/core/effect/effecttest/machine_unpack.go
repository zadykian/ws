package effecttest

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/zadykian/ws/engine/core/internal/archive"
)

// Unpack extracts the archive in Files into a temporary directory, as the real machine's does, and
// records the tree under dir in Files, Dirs, Links and Modes. It makes dir at 0755 where missing,
// and refuses what archive.Extract refuses. A tree unpacked over files already there adds to them.
func (m *Machine) Unpack(_ context.Context, file, dir string, strip int) error {
	m.mu.Lock()
	m.Calls = append(m.Calls, "unpack "+file+" "+dir)
	data, ok := m.Files[file]
	m.mu.Unlock()
	if !ok {
		return &fs.PathError{Op: "open", Path: file, Err: fs.ErrNotExist}
	}
	tmp, err := os.MkdirTemp("", "effecttest-unpack-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }() // a scratch tree
	root, err := os.OpenRoot(tmp)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }() // read back by path
	if err := archive.Extract(bytes.NewReader(data), root, strip); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Dirs[dir]; !ok {
		m.Dirs[dir] = 0o755
	}
	return filepath.WalkDir(tmp, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == tmp {
			return err
		}
		rel, err := filepath.Rel(tmp, p)
		if err != nil {
			return err
		}
		return m.record(path.Join(dir, filepath.ToSlash(rel)), p, d)
	})
}

// record stores one entry of an unpacked tree at the path.
func (m *Machine) record(to, from string, d fs.DirEntry) error {
	info, err := d.Info()
	if err != nil {
		return err
	}
	switch {
	case d.Type()&fs.ModeSymlink != 0:
		m.Links[to], err = os.Readlink(from)
	case d.IsDir():
		m.Dirs[to] = info.Mode().Perm()
	default:
		m.Files[to], err = os.ReadFile(from) //nolint:gosec // a path in the fake's own temp tree
		if perm := info.Mode().Perm(); perm != 0o644 {
			m.Modes[to] = perm
		}
	}
	return err
}
