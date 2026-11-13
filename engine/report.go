package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
)

// RoleSections turns the roles' reports into the state screen's sections.
func RoleSections(states []role.RoleState) []tui.StateSection {
	sections := make([]tui.StateSection, len(states))
	for i, s := range states {
		sections[i] = tui.StateSection{Name: s.Role, Reports: s.Steps}
	}
	return sections
}

// WriteJSON prints doc as indented JSON.
func WriteJSON(w io.Writer, doc any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

// WriteRoles prints the roles' reports as text.
func WriteRoles(w io.Writer, states []role.RoleState) error {
	var b strings.Builder
	for _, s := range states {
		fmt.Fprintf(&b, "%s\n", s.Role)
		for _, r := range s.Steps {
			WriteReport(&b, r)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// WriteReport prints one step's items and changes, indented under the step.
func WriteReport(b *strings.Builder, r step.Report) {
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

// firstLine returns the line that says what changes. The lines after it hold a file's diff, which
// the item and --json show.
func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return line
}
