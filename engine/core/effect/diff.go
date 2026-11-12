package effect

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// contextLines is the unchanged lines that diff -u keeps on each side of a change.
const contextLines = 3

// Diff returns the unified diff of the lines of before and after, as diff -u prints it with the
// same name on both sides, and "" where they are equal.
func Diff(name string, before, after []byte) string {
	return DiffAs(name, name, before, after)
}

// DiffAs is Diff with a label for each side, such as /dev/null for a file that is new.
func DiffAs(from, to string, before, after []byte) string {
	a, b := splitLines(before), splitLines(after)
	if slicesEqual(a, b) {
		return ""
	}
	// lcs[i][j] is the length of the longest common run of a[i:] and b[j:].
	lcs := make([][]int, len(a)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	// ops holds one line per entry: " ", "-" or "+" and the text.
	var ops []string
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case i < len(a) && j < len(b) && a[i] == b[j]:
			ops = append(ops, " "+a[i])
			i++
			j++
		case j < len(b) && (i == len(a) || lcs[i][j+1] > lcs[i+1][j]):
			ops = append(ops, "+"+b[j])
			j++
		default:
			ops = append(ops, "-"+a[i])
			i++
		}
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n", from, to)
	for _, h := range hunks(ops) {
		writeHunk(&sb, ops, h)
	}
	return sb.String()
}

// hunk is the range [start, end) of ops that one @@ header covers.
type hunk struct{ start, end int }

// hunks groups the changes of ops with their context. Changes at most twice the context apart share
// a hunk.
func hunks(ops []string) []hunk {
	var out []hunk
	last := -1 // the index of the last change that a hunk covers
	for k, op := range ops {
		if op[0] == ' ' {
			continue
		}
		if n := len(out); n > 0 && k-last-1 <= 2*contextLines {
			out[n-1].end = min(k+contextLines+1, len(ops))
		} else {
			out = append(out, hunk{max(k-contextLines, 0), min(k+contextLines+1, len(ops))})
		}
		last = k
	}
	return out
}

func writeHunk(sb *strings.Builder, ops []string, h hunk) {
	// Each file's first line in the hunk follows the lines of ops that it has before the hunk.
	fromStart, toStart := 1, 1
	for _, op := range ops[:h.start] {
		if op[0] != '+' {
			fromStart++
		}
		if op[0] != '-' {
			toStart++
		}
	}
	var fromCount, toCount int
	for _, op := range ops[h.start:h.end] {
		if op[0] != '+' {
			fromCount++
		}
		if op[0] != '-' {
			toCount++
		}
	}
	fmt.Fprintf(sb, "@@ -%s +%s @@\n", hunkRange(fromStart, fromCount), hunkRange(toStart, toCount))
	for _, op := range ops[h.start:h.end] {
		sb.WriteString(op + "\n")
	}
}

// hunkRange writes diff -u's range: the start alone for one line, and the line before for none.
func hunkRange(start, count int) string {
	switch count {
	case 1:
		return strconv.Itoa(start)
	case 0:
		return fmt.Sprintf("%d,0", start-1)
	}
	return fmt.Sprintf("%d,%d", start, count)
}

func splitLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	return strings.Split(string(bytes.TrimSuffix(data, []byte("\n"))), "\n")
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
