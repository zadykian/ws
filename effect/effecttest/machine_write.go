package effecttest

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/zadykian/ws/effect"
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

// RunInput records the command as Run's, and its input in Inputs, and returns its result from the table.
func (m *Machine) RunInput(_ context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	m.mu.Lock()
	m.Inputs = append(m.Inputs, string(stdin))
	m.mu.Unlock()
	return m.run("run", name, args)
}

// PassInput records the command as RunInput does, and returns the error of its result.
func (m *Machine) PassInput(ctx context.Context, stdin []byte, name string, args ...string) error {
	_, err := m.RunInput(ctx, stdin, name, args...)
	return err
}

// PassLines records the command as RunInput does, and offers each line of its result's output to
// take. A line that take leaves goes in Passed. It returns the error of its result.
func (m *Machine) PassLines(ctx context.Context, stdin []byte, take func(line string) bool, name string, args ...string) error {
	out, err := m.RunInput(ctx, stdin, name, args...)
	for line := range strings.Lines(string(out)) {
		if line = strings.TrimSuffix(line, "\n"); !take(line) {
			m.mu.Lock()
			m.Passed = append(m.Passed, line)
			m.mu.Unlock()
		}
	}
	return err
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

// PassForeground records the command as "foreground NAME ARG", or "foreground root NAME ARG", and
// returns the error of its result.
func (m *Machine) PassForeground(_ context.Context, root bool, name string, args ...string) error {
	kind := "foreground"
	if root {
		kind += " root"
	}
	_, err := m.run(kind, name, args)
	return err
}

// Exec records "exec NAME ARGS" and returns nothing, as a command that ended ws would.
func (m *Machine) Exec(name string, args, env []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ExecEnv = append(m.ExecEnv, env)
	m.Calls = append(m.Calls, "exec "+strings.Join(append([]string{name}, args...), " "))
	return nil
}

func (m *Machine) run(kind, name string, args []string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	line := strings.Join(append([]string{name}, args...), " ")
	m.Calls = append(m.Calls, kind+" "+line)
	r, ok := m.Results[line]
	if !ok && kind == "root" && name == "rm" {
		for _, a := range args {
			delete(m.Files, a)
			delete(m.Root, a)
		}
		return nil, nil
	}
	if !ok && kind == "root" && name == "mkdir" {
		return nil, nil
	}
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

// ReadDir returns the names of the files, links and directories directly under the directory.
func (m *Machine) ReadDir(path string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, "readdir "+path)
	if _, ok := m.Dirs[path]; !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}
	var names []string
	var paths []string
	for f := range m.Files {
		paths = append(paths, f)
	}
	for d := range m.Dirs {
		paths = append(paths, d)
	}
	for l := range m.Links {
		paths = append(paths, l)
	}
	for _, f := range paths {
		if dir, name := f[:strings.LastIndex(f, "/")], f[strings.LastIndex(f, "/")+1:]; dir == path {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

// Mkdir makes the directory, or fails with fs.ErrExist where the path exists.
func (m *Machine) Mkdir(path string, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, fmt.Sprintf("mkdir %s %o", path, perm))
	if _, ok := m.Dirs[path]; ok {
		return &fs.PathError{Op: "mkdir", Path: path, Err: fs.ErrExist}
	}
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
	} else if a, ok := m.Root[path]; ok {
		a.Mode = perm.Perm()
		m.Root[path] = a
	} else {
		m.Modes[path] = perm.Perm()
	}
	return nil
}

// WriteAtomic stores the file with its mode and owner.
func (m *Machine) WriteAtomic(path string, data []byte, a effect.Attrs) error {
	if err := m.write("atomic", path, data, a.Mode); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Owners[path] = [2]int{a.UID, a.GID}
	return nil
}

// Chown records the owner of the file or directory.
func (m *Machine) Chown(path string, uid, gid int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, fmt.Sprintf("chown %s %d:%d", path, uid, gid))
	m.Owners[path] = [2]int{uid, gid}
	return nil
}

// Symlink stores the link.
func (m *Machine) Symlink(target, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, "symlink "+target+" "+path)
	m.Links[path] = target
	return nil
}

// Download stores the page of Pages at the URL, or fails and stores nothing on a missing page or a
// sum that differs.
func (m *Machine) Download(_ context.Context, url, path, sum string, a effect.Attrs) error {
	m.mu.Lock()
	body, ok := m.Pages[url]
	m.Calls = append(m.Calls, "download "+url)
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("download %s: 404 Not Found", url)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(body)); got != sum {
		return fmt.Errorf("download %s: SHA-256 is %s, want %s", url, got, sum)
	}
	return m.WriteAtomic(path, body, a)
}
