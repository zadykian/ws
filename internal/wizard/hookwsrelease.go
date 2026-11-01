package wizard

import "strings"

// sumOf returns the SHA-256 that the sums, as sha256sum prints them, list for the file, in lower
// case, or an empty string where they list none.
func sumOf(sums []byte, file string) string {
	for _, l := range strings.Split(string(sums), "\n") {
		sum, name, ok := strings.Cut(strings.TrimSpace(l), " ")
		name = strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(name), "*"), "./")
		if ok && name == file {
			return strings.ToLower(sum)
		}
	}
	return ""
}

