package main

import (
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// macDoc is what ws mac state --json prints.
type macDoc struct {
	Version string        `json:"version"`
	Steps   []step.Report `json:"steps"`
}

// newMacStateCmd returns ws mac state.
func newMacStateCmd(p process) *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "state",
		Short: "report what the Mac's steps would change",
		Long: "It reads alone and applies nothing, as ws mac apply --check does, and needs what it needs: " +
			"mac.conf, and not root. The Mac's steps report their changes and no items yet, so a step " +
			"that fails stands in the report as a check. The exit code stays 0. A report holds no secret.\n\n" +
			"The report is on stdout and ws's own lines on stderr. --json prints the version of ws, then " +
			"each step with its items and changes.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error { return runMacState(c, p, asJSON) },
	}
	c.Flags().BoolVar(&asJSON, "json", false, "print the report as JSON")
	return c
}

// runMacState prints what the steps of ws mac apply report. The ssh step that runs again is left
// out: under a read it would repeat the first.
func runMacState(c *cobra.Command, p process, asJSON bool) error {
	home, settings, err := loadMacConf(p)
	if err != nil {
		return err
	}
	term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
	out := progOut{term, c.CommandPath(), map[string]bool{}}
	steps := setupSteps(settings, home, p.getenv("XDG_CONFIG_HOME"), p.machine, out, io.Discard, true)
	reports, err := step.Read(c.Context(), steps)
	term.Flush()
	if err != nil {
		return err
	}
	if asJSON {
		return writeJSON(c.OutOrStdout(), macDoc{Version: p.version, Steps: reports})
	}
	return writeMacState(c.OutOrStdout(), reports)
}

// writeMacState prints the steps' reports as text.
func writeMacState(w io.Writer, reports []step.Report) error {
	var b strings.Builder
	for _, r := range reports {
		writeReport(&b, r)
	}
	_, err := io.WriteString(w, b.String())
	return err
}
