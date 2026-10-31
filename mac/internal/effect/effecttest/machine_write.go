package effecttest

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
)

// WriteFile stores the file.
func (m *Machine) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return m.write("write", path, data, perm)
}

// WriteRootFile stores the file as root's.
func (m *Machine) WriteRootFile(path string, data []byte, perm fs.FileMode) error {
	if err := m.write("write root", path, data, perm); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Root[path] = effect.Attrs{Mode: perm.Perm()}
	return nil
}

func (m *Machine) write(kind, path string, data []byte, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, fmt.Sprintf("%s %s %o", kind, path, perm))
	m.Files[path] = append([]byte(nil), data...)
	m.Modes[path] = perm.Perm()
	return nil
}

// Run records the command and returns its result from the table.
func (m *Machine) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	return m.run("run", name, args)
}

// RunRoot records the command as root's and returns its result from the table.
func (m *Machine) RunRoot(_ context.Context, name string, args ...string) ([]byte, error) {
	return m.run("root", name, args)
}

// RunShown records the command as Run's or RunRoot's and returns its result from the table.
func (m *Machine) RunShown(_ context.Context, root bool, name string, args ...string) ([]byte, error) {
	if root {
		return m.run("root", name, args)
	}
	return m.run("run", name, args)
}

// Pass records the command as Run's or RunRoot's and returns the error of its result.
func (m *Machine) Pass(ctx context.Context, root bool, name string, args ...string) error {
	_, err := m.RunShown(ctx, root, name, args...)
	return err
}

func (m *Machine) run(kind, name string, args []string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	line := strings.Join(append([]string{name}, args...), " ")
	m.Calls = append(m.Calls, kind+" "+line)
	r, ok := m.Results[line]
	if !ok {
		return nil, fmt.Errorf("effecttest: no result for %q", line)
	}
	return r.Out, r.Err
}

// Stat describes a directory, a link or a file of the maps. A file's mode is 0644 unless Modes has it.
func (m *Machine) Stat(path string) (effect.Info, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, "stat "+path)
	info := effect.Info{Link: m.Links[path]}
	target := path
	if info.Link != "" {
		target = info.Link
	}
	if mode, ok := m.Dirs[target]; ok {
		info.Mode = mode | fs.ModeDir
		return info, nil
	}
	if _, ok := m.Files[target]; ok {
		info.Mode = 0o644
		if mode, ok := m.Modes[target]; ok {
			info.Mode = mode
		}
		return info, nil
	}
	if info.Link != "" {
		return info, nil
	}
	return effect.Info{}, &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
}

// ReadDir returns the names of the files directly under the directory.
func (m *Machine) ReadDir(path string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, "readdir "+path)
	if _, ok := m.Dirs[path]; !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}
	var names []string
	for f := range m.Files {
		if dir, name := f[:strings.LastIndex(f, "/")], f[strings.LastIndex(f, "/")+1:]; dir == path {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

// Mkdir makes the directory.
func (m *Machine) Mkdir(path string, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, fmt.Sprintf("mkdir %s %o", path, perm))
	m.Dirs[path] = perm.Perm()
	return nil
}

// Chmod sets the mode of the directory or the file.
func (m *Machine) Chmod(path string, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, fmt.Sprintf("chmod %s %o", path, perm))
	if _, ok := m.Dirs[path]; ok {
		m.Dirs[path] = perm.Perm()
	} else {
		m.Modes[path] = perm.Perm()
	}
	return nil
}
