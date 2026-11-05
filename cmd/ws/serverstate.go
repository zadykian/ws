package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/step"
)

// writeState prints the roles' reports as text.
func writeState(w io.Writer, states []apply.RoleState) error {
	var b strings.Builder
	for _, s := range states {
		fmt.Fprintf(&b, "%s\n", s.Role)
		for _, r := range s.Steps {
			writeReport(&b, r)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// writeReport prints one step's items and changes, indented under the step.
func writeReport(b *strings.Builder, r step.Report) {
	fmt.Fprintf(b, "  %s\n", r.Step)
	if r.After != "" {
		fmt.Fprintf(b, "    not read: after %s\n", r.After)
	}
	for _, it := range r.Items {
		fmt.Fprintf(b, "    %s %s: found %s, want %s\n", it.Kind, it.Name, it.Found, it.Want)
		if it.Conflict != "" {
			fmt.Fprintf(b, "      conflict: %s\n", it.Conflict)
		}
		if it.Diff != "" {
			for line := range strings.SplitSeq(strings.TrimSuffix(it.Diff, "\n"), "\n") {
				fmt.Fprintf(b, "      %s\n", line)
			}
		}
	}
	for _, ch := range r.Changes {
		fmt.Fprintf(b, "    pending: %s\n", firstLine(ch.Text))
	}
}

// firstLine returns text up to its first line break: a file's change holds its diff, which the
// item shows.
func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return line
}
