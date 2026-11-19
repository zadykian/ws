package values

import (
	"fmt"
	"slices"
	"strings"
)

// Set returns the file data with key name at value: its line in place, or a new line at the end,
// and every other line unchanged but a follower's. An empty value takes the key's line out, which
// gives its default back. A follower's line that holds the value derive gives it from the rest of
// the file moves to the value derive gives from the new file. One set by hand stays. Set returns
// the names of the followers it moved. It refuses a key that a line does not hold, and a result
// that does not read as value.
func Set(data []byte, name, value string) ([]byte, []string, error) {
	keys := Keys()
	at := slices.IndexFunc(keys, func(k Key) bool { return k.Name == name })
	if at < 0 {
		return nil, nil, fmt.Errorf("no line holds %s: edit the file by hand", name)
	}
	key := keys[at]
	if err := key.Check(value); err != nil {
		return nil, nil, err
	}
	if _, err := Parse(data); err != nil {
		return nil, nil, err
	}
	text := string(data)
	if strings.TrimSpace(text) == "" {
		text = "---\n"
	}
	lines := strings.SplitAfter(strings.TrimSuffix(text, "\n")+"\n", "\n")
	lines = lines[:len(lines)-1]
	var moving []Key
	for _, f := range followers {
		k := keys[slices.IndexFunc(keys, func(k Key) bool { return k.Name == f })]
		if had, want, ok := derived(lines, k); ok && f != name && had == want {
			moving = append(moving, k)
		}
	}
	start, end := span(lines, name)
	line := ""
	if value == "" {
		if start == len(lines) {
			return data, nil, nil
		}
		lines = slices.Delete(lines, start, end)
	} else {
		var err error
		if line, err = key.line(value); err != nil {
			return nil, nil, err
		}
		lines = slices.Replace(lines, start, end, line)
	}
	var moved []string
	for _, k := range moving {
		if had, want, ok := derived(lines, k); ok && had != want {
			follower, err := k.line(want)
			if err != nil {
				return nil, nil, err
			}
			start, end := span(lines, k.Name)
			lines = slices.Replace(lines, start, end, follower)
			moved = append(moved, k.Name)
		}
	}
	next, err := checked([]byte(strings.Join(lines, "")), key, line)
	return next, moved, err
}

// derived returns the value of k that lines give, and the one that derive gives it without k's
// line. ok is false where no line holds k, or where lines read no more without it, as where it
// ends a document. Such a line stays, and checked judges the result.
func derived(lines []string, k Key) (had, want string, ok bool) {
	start, end := span(lines, k.Name)
	v, err := decode([]byte(strings.Join(lines, "")))
	w, errw := decode([]byte(strings.Join(slices.Delete(slices.Clone(lines), start, end), "")))
	if start == len(lines) || err != nil || errw != nil {
		return "", "", false
	}
	return v.Text(k), w.Text(k), true
}

// checked returns data where the file reads and gives key the value of line, which is empty for a
// key taken out. The line-by-line edit knows no YAML: this catches a file that it misread, such as
// one whose document ends before the new line.
func checked(data []byte, key Key, line string) ([]byte, error) {
	got, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("the file reads no more with %s changed: edit it by hand: %w", key.Name, err)
	}
	want, err := decode([]byte(line))
	if err != nil {
		return nil, err
	}
	if line != "" && got.Text(key) != want.Text(key) {
		return nil, fmt.Errorf("the file does not give the new %s: edit it by hand", key.Name)
	}
	return data, nil
}

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
