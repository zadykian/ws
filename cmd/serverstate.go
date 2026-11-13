package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/server/ubuntu/apply"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

// newStateCmd returns ws server ubuntu state.
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
			"At a terminal it draws the state screen on stderr, where the tabs, a filter and r to read again are. " +
			"--plain prints the report as text, and so does a pipe. The text is on stdout and ws's own lines on stderr. --json prints the version of ws, then " +
			"each role with its steps, items and changes.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error { return runState(c, p, applyOptions{Options: opts}, asJSON) },
	}
	f := c.Flags()
	f.BoolVar(&asJSON, "json", false, "print the report as JSON")
	f.StringSliceVar(&opts.Only, "only", nil, "report the named roles and ROLE/STEPs alone, as a list or by repeating the flag")
	return c
}

// runState prints what the roles report.
func runState(c *cobra.Command, p process, opts applyOptions, asJSON bool) error {
	r, err := prepareServer(c, p, opts)
	if err != nil {
		return err
	}
	// State reads the machine and takes no secrets. A step that needs them would write them out to
	// plan, as ikeward's install does, and a report has no use for them.
	r.cfg.Secrets = secrets.Secrets{Unread: true}
	if drawsReport(c, p) {
		p.machine = r.cfg.M // ws_home confines it.
		return tui.RunState(c.Context(), screenConfig(c, p), c.CommandPath(), func(ctx context.Context, b *tui.Bridge) ([]tui.StateSection, error) {
			// The read takes the screen's effects as a run does, as the steps don't own the terminal under it.
			states, err := apply.State(ctx, onScreen(r.cfg, b), r.list)
			return roleSections(states), err
		})
	}
	states, err := apply.State(c.Context(), r.cfg, r.list)
	r.term.Flush()
	if err != nil {
		return err
	}
	if asJSON {
		return writeJSON(c.OutOrStdout(), apply.StateDoc{Version: p.version, Roles: states})
	}
	return writeState(c.OutOrStdout(), states)
}

// roleSections turns the roles' reports into the state screen's sections.
func roleSections(states []apply.RoleState) []tui.StateSection {
	sections := make([]tui.StateSection, len(states))
	for i, s := range states {
		sections[i] = tui.StateSection{Name: s.Role, Reports: s.Steps}
	}
	return sections
}

// writeJSON prints doc as indented JSON.
func writeJSON(w io.Writer, doc any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
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

// firstLine returns the line that says what changes. The lines after it hold a file's diff, which
// the item and --json show.
func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return line
}
