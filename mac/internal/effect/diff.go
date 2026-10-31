package effect

import (
	"bytes"
	"fmt"
	"strings"
)

// Diff returns the unified diff of the lines of before and after, in one hunk, and "" where they are
// equal.
func Diff(name string, before, after []byte) string {
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
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n@@ -1,%d +1,%d @@\n", name, name, len(a), len(b))
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case i < len(a) && j < len(b) && a[i] == b[j]:
			sb.WriteString(" " + a[i] + "\n")
			i++
			j++
		case j < len(b) && (i == len(a) || lcs[i][j+1] > lcs[i+1][j]):
			sb.WriteString("+" + b[j] + "\n")
			j++
		default:
			sb.WriteString("-" + a[i] + "\n")
			i++
		}
	}
	return sb.String()
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
