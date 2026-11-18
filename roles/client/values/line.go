package values

import (
	"regexp"
	"strconv"
	"strings"
)

// bare matches a value that YAML reads back as itself after a key and a colon. It has no reserved
// first character (@ and % included), no # or :, and no blank last.
var bare = regexp.MustCompile(`^[A-Za-z0-9_./=+~][A-Za-z0-9_./=+@~%, -]*$`)

// Line is the key's line for a values file, as YAML: the value bare where YAML reads it back as
// the same text, else in double quotes. An empty value is "", and ~ and null are quoted, as YAML
// reads them as nothing. The values are ASCII, where Go's escapes and YAML's agree.
func Line(key, value string) string {
	switch {
	case bare.MatchString(value) && !strings.HasSuffix(value, " ") && !isNull(value):
		return key + ": " + value
	default:
		return key + ": " + strconv.Quote(value)
	}
}

// isNull reports whether YAML reads a bare v as the empty value.
func isNull(v string) bool {
	return v == "~" || v == "null" || v == "Null" || v == "NULL"
}

// Text returns the lines that give each key its value, in the order given: Text(KEY, VALUE, ...).
// Each line is as Line writes it, and ends in a newline. An odd count panics, as
// strings.NewReplacer does, for a caller has slipped.
func Text(pairs ...string) string {
	if len(pairs)%2 != 0 {
		panic("values.Text: a key has no value")
	}
	var b strings.Builder
	for i := 1; i < len(pairs); i += 2 {
		b.WriteString(Line(pairs[i-1], pairs[i]))
		b.WriteByte('\n')
	}
	return b.String()
}
