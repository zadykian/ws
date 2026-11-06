package values

import (
	"slices"
	"strings"
)

// span finds the lines from start to end that hold key name, with the lines that continue its
// value. Blank lines and comments after the value stay, as they lead to the next key. Without a
// line for the key, both are len(lines).
func span(lines []string, name string) (start, end int) {
	start = slices.IndexFunc(lines, func(line string) bool {
		rest, ok := strings.CutPrefix(line, name)
		return ok && strings.HasPrefix(strings.TrimLeft(rest, " \t"), ":")
	})
	if start < 0 {
		return len(lines), len(lines)
	}
	end = start + 1
	for end < len(lines) && (spare(lines[end]) || strings.HasPrefix(lines[end], "- ") ||
		strings.HasPrefix(lines[end], " ") || strings.HasPrefix(lines[end], "\t")) {
		end++
	}
	for end > start+1 && spare(lines[end-1]) {
		end--
	}
	return start, end
}

// spare says that a line is blank, or a comment.
func spare(line string) bool {
	line = strings.TrimSpace(line)
	return line == "" || strings.HasPrefix(line, "#")
}
