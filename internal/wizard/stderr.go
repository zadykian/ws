package wizard

import (
	"regexp"
	"strings"
)

// ExitStatus is what the machine's error holds before the stderr of a command.
var ExitStatus = regexp.MustCompile(`^.*exit status \d+: `)

// Stderr returns the stderr that a failed command's error holds, in one line, without its argv.
func Stderr(err error) string {
	var lines []string
	for _, l := range strings.Split(ExitStatus.ReplaceAllString(err.Error(), ""), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return strings.Join(lines, "; ")
}
