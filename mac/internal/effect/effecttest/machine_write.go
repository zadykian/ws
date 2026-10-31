package effecttest

import (
	"context"
	"fmt"
	"io/fs"
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
