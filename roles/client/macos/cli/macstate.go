package cli

import (
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/step"
)

// macDoc is what ws client macos state --json prints.
type macDoc struct {
	Version string        `json:"version"`
	Steps   []step.Report `json:"steps,omitempty"`
	Servers []serverDoc   `json:"servers,omitempty"`
}

// serverDoc is one server's steps, where the report covers several servers.
type serverDoc struct {
	Name  string        `json:"name"`
	Steps []step.Report `json:"steps"`
}

// newMacStateCmd returns ws client macos state.
func newMacStateCmd(p enginecli.Env, server *string) *cobra.Command {
	c := &cobra.Command{
		Use:   "state",
		Short: "report what the Mac's steps would change",
		Long: "It reads alone and applies nothing, as ws client macos apply --check does, and needs what it needs: " +
			"values.yml, and not root. The Mac's steps report the packages, services and files ws manages " +
			"there, and a step that fails stands in the report as a check. The exit code stays 0. " +
			"With several servers, one whose file ws refuses stands in the report with the refusal. " +
			"A report holds no secret.\n\n" +
			"The text is on stdout and ws's own lines on stderr. --json prints the version of ws, then " +
			"each step with its items and changes.",
		Args: cobra.NoArgs,
		RunE: enginecli.State(macTarget(p, new([]string), server), p),
	}
	c.Flags().Bool("json", false, "print the report as JSON")
	return c
}

// writeMacState prints the steps' reports as text, under a line of the server's name where the
// report covers several servers.
func writeMacState(w io.Writer, docs []serverDoc, grouped bool) error {
	var b strings.Builder
	for _, d := range docs {
		if grouped {
			b.WriteString(d.Name + "\n")
		}
		for _, r := range d.Steps {
			engine.WriteReport(&b, r)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}
