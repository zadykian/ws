package conf

import "strings"

// Assignments prints the settings as shell assignments, a line a key, each value in single quotes,
// for a shell to evaluate in place of mac.conf, which it never runs.
func (c Conf) Assignments() string {
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k.name + "='" + strings.ReplaceAll(*k.field(&c), "'", `'\''`) + "'\n")
	}
	return b.String()
}
