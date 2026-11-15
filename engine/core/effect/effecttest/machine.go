package effecttest

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/zadykian/ws/engine/core/effect"
)

// Result is what a command returns.
type Result struct {
	Out []byte
	Err error
}

// Machine keeps files in a map and answers commands from a table.
type Machine struct {
	mu      sync.Mutex
	Files   map[string][]byte
	Root    map[string]effect.Attrs // the files root owns: ReadFile cannot read them
	Results map[string]Result       // by the command line, as "name arg arg"
	Dirs    map[string]fs.FileMode  // the directories, with their modes
	Links   map[string]string       // the links, to the paths they point to
	Modes   map[string]fs.FileMode  // the modes set on files, where not 0644
	Pages   map[string][]byte       // the bodies Download serves, by URL
	Owners  map[string][2]int       // the uid and gid set on files and directories, where not root's
	Inputs  []string                // the standard input of each RunInput
	Passed  []string                // the lines that PassLines passed on
	ExecEnv [][]string              // the env of each Exec
	Calls   []string                // each call, as "run NAME ARG", "root NAME ARG", "write PATH" ...

	// Packages is dpkg's database. A non-nil table answers dpkg-query once Results holds no line.
	Packages Packages
}

// NewMachine returns an empty Machine.
func NewMachine() *Machine {
	return &Machine{Files: map[string][]byte{}, Root: map[string]effect.Attrs{}, Results: map[string]Result{},
		Dirs: map[string]fs.FileMode{}, Links: map[string]string{}, Modes: map[string]fs.FileMode{},
		Owners: map[string][2]int{}, Pages: map[string][]byte{}}
}

// ReadFile returns the file, or fs.ErrNotExist.
func (m *Machine) ReadFile(path string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, "read "+path)
	if _, ok := m.Root[path]; ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrPermission}
	}
	data, ok := m.Files[path]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}
	return data, nil
}

// ReadRootFile returns the file, or fs.ErrNotExist.
func (m *Machine) ReadRootFile(path string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, "read root "+path)
	data, ok := m.Files[path]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}
	return data, nil
}

// ReadHomeFile is ReadRootFile with RootAttrs, as the real machine's ReadHomeFile reads. It fails
// on a path not below home, on a link at the path or on the way, and on a directory. It fails on a
// file of more than effect.HomeFileMax bytes too.
func (m *Machine) ReadHomeFile(home, path string) ([]byte, effect.Attrs, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, "read home "+path)
	rel, err := filepath.Rel(home, path)
	if err != nil || !filepath.IsLocal(rel) {
		return nil, effect.Attrs{}, fmt.Errorf("%s is not a path below the home %s", path, home)
	}
	for at := path; at != home; at = filepath.Dir(at) {
		if _, linked := m.Links[at]; linked {
			return nil, effect.Attrs{}, &fs.PathError{Op: "open", Path: path, Err: syscall.ELOOP}
		}
	}
	if _, isDir := m.Dirs[path]; isDir {
		return nil, effect.Attrs{}, &fs.PathError{Op: "read", Path: path, Err: fmt.Errorf("%w: d", effect.ErrNotRegular)}
	}
	data, ok := m.Files[path]
	if !ok {
		return nil, effect.Attrs{}, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}
	if len(data) > effect.HomeFileMax {
		return nil, effect.Attrs{}, &fs.PathError{Op: "read", Path: path, Err: effect.ErrTooLarge}
	}
	a, _ := m.rootAttrs(path)
	return data, a, nil
}

// RootAttrs returns the attributes of a root file, a directory or a file. A file put in Files alone
// is root's 0:0, 0644.
func (m *Machine) RootAttrs(path string) (effect.Attrs, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, "stat root "+path)
	return m.rootAttrs(path)
}

// rootAttrs is RootAttrs under the lock.
func (m *Machine) rootAttrs(path string) (effect.Attrs, error) {
	a, ok := m.Root[path]
	if !ok {
		if mode, isDir := m.Dirs[path]; isDir {
			a, ok = effect.Attrs{Mode: mode}, true
		} else if _, isFile := m.Files[path]; isFile {
			a, ok = effect.Attrs{Mode: 0o644}, true
			if mode, set := m.Modes[path]; set {
				a.Mode = mode
			}
		}
	}
	if !ok {
		return effect.Attrs{}, &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
	}
	if o, set := m.Owners[path]; set {
		a.UID, a.GID = o[0], o[1]
	}
	return a, nil
}
