// Command vale-comments runs Vale on the comments of the files written here, which Vale reads
// whole, code and strings included, or not at all:
//
//	go run ./tools/vale-comments [PATHSPEC...]
//
// It mirrors the comments of each file that git lists for the PATHSPECs, or of every file, as
// Markdown. It runs the pinned Vale with .vale.ini on the mirrors, and prints each alert at its
// source, as FILE:LINE:COL CHECK MESSAGE. It exits with 1 where it prints one, and with 2 where it
// fails, though go run exits with 1 for both. Run it from the repository's root.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(pathspecs []string, stdout, stderr io.Writer) int {
	files, err := listed(pathspecs)
	var found []finding
	if err == nil {
		found, err = check(files, stderr)
	}
	for _, f := range found {
		_, _ = fmt.Fprintf(stdout, "%s:%d:%d %s %s\n", f.file, f.line, f.col, f.check, f.message)
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "vale-comments: %v\n", err)
		return 2
	}
	return min(len(found), 1)
}

// listed returns the files that git lists for the pathspecs, new ones included but not those it
// ignores. It fails where git lists none, so that no run passes having checked nothing.
func listed(pathspecs []string) ([]string, error) {
	args := append([]string{"ls-files", "-z", "--cached", "--others", "--exclude-standard", "--"}, pathspecs...)
	out, err := exec.Command("git", args...).Output() //nolint:gosec // the pathspecs go to git alone, after --
	if err != nil {
		return nil, fmt.Errorf("git cannot list the files: %w", err)
	}
	var files []string
	for file := range strings.SplitSeq(string(out), "\x00") {
		if info, err := os.Lstat(file); file != "" && err == nil && info.Mode().IsRegular() {
			files = append(files, file)
		}
	}
	if len(files) == 0 {
		return nil, errors.New("git lists no file for the pathspecs")
	}
	slices.Sort(files)
	return slices.Compact(files), nil
}
