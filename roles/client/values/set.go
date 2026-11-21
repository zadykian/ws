package values

import "strings"

// Set returns the text with each key given its value, as Line writes it: Set(text, KEY, VALUE, ...).
// A key's line is replaced where the text has one, and else added at the end. Every other line
// stays, comments and blank lines included. An odd count panics, as Text does.
func Set(text string, pairs ...string) string {
	if len(pairs)%2 != 0 {
		panic("values.Set: a key has no value")
	}
	lines := strings.SplitAfter(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for i := 1; i < len(pairs); i += 2 {
		key, line := pairs[i-1], Line(pairs[i-1], pairs[i])+"\n"
		at := -1
		for j, l := range lines {
			if k, ok := lineKey(l); ok && k == key {
				at = j
				break
			}
		}
		switch {
		case at >= 0:
			lines[at] = line
		case len(lines) > 0 && !strings.HasSuffix(lines[len(lines)-1], "\n"):
			lines[len(lines)-1] += "\n"
			fallthrough
		default:
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "")
}
