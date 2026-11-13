package netplan

import (
	"fmt"
	"strings"
)

// Result is what a command left: its exit code, and its standard error. Msg is why it did not run,
// where it didn't.
type Result struct {
	RC     int
	Stdout string
	Stderr string
	Msg    string
}

// errorLine returns the line of netplan's error, with the scratch root's path cut out. The
// warnings come before it, and the lines below it quote the line of the file at fault, which may
// hold a secret.
func errorLine(r Result, root string) string {
	for _, line := range strings.Split(r.Stderr, "\n") {
		if strings.HasPrefix(line, "Command failed: ") || strings.HasPrefix(line, "ERROR") || strings.HasPrefix(line, root) {
			line = strings.TrimPrefix(line, "Command failed: ")
			return strings.TrimRight(strings.ReplaceAll(line, root, ""), ":. \t\r")
		}
	}
	if r.Msg != "" {
		return r.Msg
	}
	return fmt.Sprintf("netplan exited with %d", r.RC)
}
