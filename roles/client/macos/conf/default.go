package conf

import "strings"

// Default gives the shell role's default of that name, as the role's file has it, or nothing.
func Default(defaults []byte, name string) string {
	for _, l := range strings.Split(string(defaults), "\n") {
		if v, ok := strings.CutPrefix(l, name+":"); ok {
			return strings.TrimLeft(v, " ")
		}
	}
	return ""
}
