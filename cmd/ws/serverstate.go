package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/step"
)

// stateDoc is what ws server state --json prints.
type stateDoc struct {
	Version string            `json:"version"`
	Roles   []apply.RoleState `json:"roles"`
}

// newStateCmd returns ws server state.
func newStateCmd(p process) *cobra.Command {
	var opts apply.Options
	var asJSON bool
	c := &cobra.Command{
		Use:   "state",
		Short: "report what the server holds of what ws manages, and what apply would change",
		Long: "It reads alone and applies nothing. It needs what --check needs: root, on Ubuntu 26.04. " +
			"It reports what ws manages and not the rest of the machine: the packages, services and files of the " +
			"steps, what stands against them, and the changes that apply would make. A step that fails stands " +
			"in the report as a check, and the exit code stays 0. It takes no secrets, so a report holds none and " +
			"says nothing of what they write.\n\n" +
			"The report is on stdout and ws's own lines on stderr. --json prints the version of ws, then " +
			"each role with its steps, items and changes.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error { return runState(c, p, opts, asJSON) },
	}
	f := c.Flags()
	f.BoolVar(&asJSON, "json", false, "print the report as JSON")
	f.StringSliceVar(&opts.Only, "only", nil, "report the named roles alone, as a list or by repeating the flag")
	return c
}

// runState prints what the roles report.
func runState(c *cobra.Command, p process, opts apply.Options, asJSON bool) error {
	r, err := prepareServer(c, p, opts)
	if err != nil {
		return err
	}
	states, err := apply.State(c.Context(), r.cfg, r.list)
	r.term.Flush()
	if err != nil {
		return err
	}
	if asJSON {
		enc := json.NewEncoder(c.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(stateDoc{Version: p.version, Roles: states})
	}
	return writeState(c.OutOrStdout(), states)
}

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
