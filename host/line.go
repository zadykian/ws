package host

import (
	"context"
	"errors"
	"io/fs"
	"regexp"
	"strings"

	"github.com/zadykian/ws/effect"
)

// Line is one line of a file that may hold others. The file is made, with the owner and mode, where
// it is missing.
type Line struct {
	M    effect.Machine
	Path string
	Text string
	// Match is a regular expression for the line Text replaces. Without it, or where no line
	// matches, Text is appended unless a line is Text already.
	Match string
	Mode  fs.FileMode
	// UID and GID are the owner, root's by default.
	UID, GID int
}

// Plan edits the file's lines and plans the file as File does.
func (l Line) Plan(ctx context.Context) ([]Change, error) {
	re, err := regexp.Compile(l.Match)
	if err != nil {
		return nil, err
	}
	before, err := l.M.ReadRootFile(l.Path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return File{M: l.M, Path: l.Path, Content: l.edit(re, before), Mode: l.Mode, UID: l.UID, GID: l.GID,
		edit: func(now []byte) []byte { return l.edit(re, now) }}.Plan(ctx)
}

// edit returns the file's content with the line replaced or appended.
func (l Line) edit(re *regexp.Regexp, before []byte) []byte {
	lines := strings.SplitAfter(string(before), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	at := -1
	for i, line := range lines {
		if strings.TrimSuffix(line, "\n") == l.Text || (l.Match != "" && re.MatchString(line)) {
			at = i
			break
		}
	}
	if at >= 0 {
		lines[at] = l.Text + "\n"
	} else {
		if n := len(lines); n > 0 && !strings.HasSuffix(lines[n-1], "\n") {
			lines[n-1] += "\n"
		}
		lines = append(lines, l.Text+"\n")
	}
	return []byte(strings.Join(lines, ""))
}
