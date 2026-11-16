package netplan

import (
	"regexp"
	"strings"
)

// globMatches reports whether name fits the glob as Python's fnmatchcase tells it, which the
// netplan check kept: * is any run, ? any one character, [seq] one of seq and [!seq] none of it.
// networkd reads these globs, and a name holds no slash.
func globMatches(glob, name string) bool {
	re, err := regexp.Compile(`(?s)^` + globRegexp(glob) + `$`)
	return err == nil && re.MatchString(name)
}

func globRegexp(glob string) string {
	var out strings.Builder
	runes := []rune(glob)
	for i := 0; i < len(runes); i++ {
		switch c := runes[i]; c {
		case '*':
			out.WriteString(".*")
		case '?':
			out.WriteString(".")
		case '[':
			end := i + 1
			if end < len(runes) && runes[end] == '!' {
				end++
			}
			if end < len(runes) && runes[end] == ']' {
				end++
			}
			for end < len(runes) && runes[end] != ']' {
				end++
			}
			if end >= len(runes) {
				out.WriteString(`\[`)
				continue
			}
			set := strings.ReplaceAll(string(runes[i+1:end]), `\`, `\\`)
			switch {
			case strings.HasPrefix(set, "!"):
				set = "^" + set[1:]
			case strings.HasPrefix(set, "^"):
				set = `\` + set
			}
			out.WriteString("[" + set + "]")
			i = end
		default:
			out.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	return out.String()
}
