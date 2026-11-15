package values

import "strings"

// Line is the key's line for mac.conf, the value in quotes where it holds a blank.
func Line(key, value string) string {
	if strings.ContainsAny(value, " \t") {
		return key + `="` + value + `"`
	}
	return key + "=" + value
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
