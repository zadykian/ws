package values

import (
	"fmt"
	"slices"
	"strings"
)

// Set returns the file data with key name at value: its line in place, or a new line at the end,
// and every other line unchanged. An empty value takes the key's line out, which gives its default
// back. It refuses a key that a line does not hold, and a result that does not read as value.
func Set(data []byte, name, value string) ([]byte, error) {
	keys := Keys()
	at := slices.IndexFunc(keys, func(k Key) bool { return k.Name == name })
	if at < 0 {
		return nil, fmt.Errorf("no line holds %s: edit the file by hand", name)
	}
	key := keys[at]
	if err := key.Check(value); err != nil {
		return nil, err
	}
	if _, err := Parse(data); err != nil {
		return nil, err
	}
	text := string(data)
	if strings.TrimSpace(text) == "" {
		text = "---\n"
	}
	lines := strings.SplitAfter(strings.TrimSuffix(text, "\n")+"\n", "\n")
	lines = lines[:len(lines)-1]
	start, end := span(lines, name)
	if value == "" {
		if start == len(lines) {
			return data, nil
		}
		return checked([]byte(strings.Join(slices.Delete(lines, start, end), "")), key, "")
	}
	line, err := key.line(value)
	if err != nil {
		return nil, err
	}
	return checked([]byte(strings.Join(slices.Replace(lines, start, end, line), "")), key, line)
}

// Named tells whether a line of data holds key name, whatever its value.
func Named(data []byte, name string) bool {
	lines := strings.SplitAfter(string(data), "\n")
	start, _ := span(lines, name)
	return start < len(lines)
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
