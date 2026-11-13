package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// valeCommand runs the pinned Vale, which .tools/run fetches on its first run. A test swaps it.
var valeCommand = []string{".tools/run", "vale"}

// check returns the alerts of Vale, with .tools/lint/vale.ini, on the comments of the files.
func check(files []string, stderr io.Writer) ([]finding, error) {
	dir, err := os.MkdirTemp("", "vale-comments")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	mirrors := map[string]*mirror{}
	for _, file := range files {
		path := filepath.Join(dir, strconv.Itoa(len(mirrors))+".md")
		m, err := mirrorFile(path, file)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		} else if m != nil {
			mirrors[path] = m
		}
	}
	if len(mirrors) == 0 {
		return nil, nil
	}
	args := slices.Concat(valeCommand[1:], []string{"--config=.tools/lint/vale.ini", "--output=JSON"}, slices.Sorted(maps.Keys(mirrors)))
	cmd := exec.Command(valeCommand[0], args...) //nolint:gosec // the command is the pinned Vale's
	cmd.Stderr = stderr
	out, err := cmd.Output()
	// Vale exits with 1 where it finds an alert of the level error.
	if exit := (*exec.ExitError)(nil); err != nil && (!errors.As(err, &exit) || exit.ExitCode() != 1) {
		return nil, fmt.Errorf("vale: %w", err)
	}
	return findings(out, mirrors)
}

// mirrorFile writes the mirror of the file's comments at path, or returns nil where it holds none.
func mirrorFile(path, file string) (*mirror, error) {
	if unchecked.MatchString(file) {
		return nil, nil
	}
	src, err := os.ReadFile(file) //nolint:gosec // the file is one the caller lists
	if err != nil || bytes.IndexByte(src, 0) >= 0 {
		return nil, err
	}
	m, err := mirrorOf(file, src)
	if err != nil || m.texts == 0 {
		return nil, err
	}
	m.file = file
	return m, os.WriteFile(path, []byte(strings.Join(m.lines, "\n")+"\n"), 0o600)
}
