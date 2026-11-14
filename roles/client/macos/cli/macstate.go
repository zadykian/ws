package cli

import (
	"context"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
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
			"mac.conf, and not root. The Mac's steps report the packages, services and files ws manages " +
			"there, and a step that fails stands in the report as a check. The exit code stays 0. " +
			"A report holds no secret.\n\n" +
			"At a terminal it draws the state screen on stderr, where sudo's password comes through a question. " +
			"--plain prints the report as text, and so does a pipe. " +
			"The text is on stdout and ws's own lines on stderr. --json prints the version of ws, then " +
			"each step with its items and changes.",
		Args: cobra.NoArgs,
		RunE: enginecli.State(macTarget(p, new([]string), server), p),
	}
	c.Flags().Bool("json", false, "print the report as JSON")
	return c
}

// macRead is the state screen's read of the Mac. It takes mac.conf itself, as the wizard opens the
// screen on a Mac that may have none: the header then says so, and the wizard stays.
func macRead(p enginecli.Env, server *string) tui.StateReader {
	return func(ctx context.Context, b *tui.Bridge) ([]tui.StateSection, error) {
		confs, grouped, err := stateConfs(p, server)
		if err != nil {
			return nil, err
		}
		var sections []tui.StateSection
		for _, sc := range confs {
			// The read takes the screen's effects as a run does, as the steps don't own the terminal under it.
			reports, err := macOf(p, sc.Name, sc.Conf, b.Effects().M, b, nil).State(ctx)
			section := tui.StateSection{Reports: reports}
			if grouped {
				section.Name = sc.Name
			}
			sections = append(sections, section)
			if err != nil {
				return sections, err
			}
		}
		return sections, nil
	}
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
