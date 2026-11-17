package values

import (
	"fmt"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

// entry is one KEY: value pair of a values file, with the line of its key. A scalar `null`, `~` or
// nothing is the empty value, as a missing line is. A pair that cannot be a setting carries err.
type entry struct {
	key, value, err string
	line            int
}

// keyLine is the form of a line that holds a key: a plain key at the line's start, with its colon
// right after it. The reader and every finder of a key share it, as the writers edit that form alone.
var keyLine = regexp.MustCompile(`^([A-Za-z0-9_][A-Za-z0-9_.-]*):(?:[ \t\r\n]|$)`)

// lineKey returns the key that the line holds, and whether it holds one.
func lineKey(line string) (string, bool) {
	m := keyLine.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// oneLine is the refusal of a line that is no KEY: value pair on one line.
const oneLine = "write one KEY: value a line, the key first and its colon right after it"

// syntaxLine strips the line that a YAML syntax error names: the text parsed is one line.
var syntaxLine = regexp.MustCompile(`^yaml: (?:line \d+: )?(.*)$`)

// entries reads the text as one KEY: value pair a line, in the order of the file, with blank lines
// and comments between. A line of any other form comes back as the one error, with its line. Those
// forms are a quoted key, a space before the colon, a flow map, `? KEY`, a BOM, a value wrapped over
// lines, and a --- or ... marker. The pairs are those that replace and Keep find, by
// lineKey. A key given twice, or a value that is no scalar, comes back as the entry's err: the
// caller checks the key first.
func entries(text string) ([]entry, []Error) {
	var out []entry
	seen := map[string]int{}
	for i, line := range strings.Split(text, "\n") {
		n := i + 1
		line = strings.TrimSuffix(line, "\r")
		if t := strings.TrimSpace(line); t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		key, ok := lineKey(line)
		if !ok {
			return nil, []Error{{n, oneLine}}
		}
		var doc yaml.Node
		if err := yaml.Unmarshal([]byte(line), &doc); err != nil {
			return nil, []Error{{n, "not YAML: " + syntaxLine.ReplaceAllString(err.Error(), "$1")}}
		}
		root := doc.Content[0]
		if root.Kind != yaml.MappingNode || len(root.Content) != 2 {
			return nil, []Error{{n, oneLine}}
		}
		v := root.Content[1]
		e := entry{key: key, line: n}
		switch {
		case v.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0:
			return nil, []Error{{n, oneLine}}
		case seen[key] != 0:
			e.err = fmt.Sprintf("%s is set already, on line %d", key, seen[key])
		case v.Kind != yaml.ScalarNode:
			e.err = key + " takes one value, not a list or a table"
		case v.Tag != "!!null":
			e.value = v.Value
		}
		if seen[key] == 0 {
			seen[key] = n
		}
		out = append(out, e)
	}
	return out, nil
}
