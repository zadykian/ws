package effecttest

import (
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
)

// WriteNew stores the file as WriteAtomic does, or fails with fs.ErrExist where a file, a
// directory or a link stands at the path.
func (m *Machine) WriteNew(path string, data []byte, a effect.Attrs) error {
	m.mu.Lock()
	_, file := m.Files[path]
	_, dir := m.Dirs[path]
	_, link := m.Links[path]
	m.mu.Unlock()
	if file || dir || link {
		return &fs.PathError{Op: "write", Path: path, Err: fs.ErrExist}
	}
	return m.WriteAtomic(path, data, a)
}
