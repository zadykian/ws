package fans

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Sysfs is the files of /sys.
type Sysfs interface {
	// Read returns the text of the file, without the space around it.
	Read(path string) (string, error)
	// Glob returns the paths that match, in order.
	Glob(pattern string) []string
}

// OSSysfs is the machine's /sys. A file that fails is a MissingError: files come and go.
type OSSysfs struct{}

// reason returns why a file operation failed, as the system words it.
func reason(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	return err.Error()
}

// Read reads the file.
func (OSSysfs) Read(path string) (string, error) {
	b, err := os.ReadFile(path) //nolint:gosec // a sysfs path that Hwmon built
	if err != nil {
		return "", missing("cannot read %s: %s", path, reason(err))
	}
	return strings.TrimSpace(string(b)), nil
}

// Glob lists the paths that match, sorted.
func (OSSysfs) Glob(pattern string) []string {
	paths, _ := filepath.Glob(pattern) // the pattern is ours, so it is well formed
	sort.Strings(paths)
	return paths
}
