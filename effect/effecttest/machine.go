package effecttest

import (
	"io/fs"
	"sync"

	"github.com/zadykian/ws/effect"
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
	Owners  map[string][2]int       // the uid and gid set on files and directories, where not root's
	Inputs  []string                // the standard input of each RunInput
	ExecEnv [][]string              // the env of each Exec
	Calls   []string                // each call, as "run NAME ARG", "root NAME ARG", "write PATH" ...
}

// NewMachine returns an empty Machine.
func NewMachine() *Machine {
	return &Machine{Files: map[string][]byte{}, Root: map[string]effect.Attrs{}, Results: map[string]Result{},
		Dirs: map[string]fs.FileMode{}, Links: map[string]string{}, Modes: map[string]fs.FileMode{},
		Owners: map[string][2]int{}}
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

// RootAttrs returns the attributes of a root file, a directory or a file. A file put in Files alone
// is root's 0:0, 0644.
func (m *Machine) RootAttrs(path string) (effect.Attrs, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, "stat root "+path)
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
