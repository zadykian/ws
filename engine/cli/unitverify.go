package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/core/kind"
)

// manager is mise's package manager for ws's units: the plugin's name, which prefixes each unit in
// [bootstrap.packages].
const manager = "ws"

// miseStatus is what mise bootstrap packages status --json prints: each package manager by name,
// whether mise loaded it, and the packages that the config asks of it.
type miseStatus map[string]struct {
	Available bool `json:"available"`
	Packages  []struct {
		Package string `json:"package"`
	} `json:"packages"`
}

// unitVerify returns ws unit verify.
func unitVerify(p Env, s UnitSide) *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "fail where mise's plan lacks a unit of the specs",
		Long: "It reads mise's packages, by mise bootstrap packages status --json, and fails where the ws " +
			"plugin is not loaded or a unit of the specs is not among its packages: mise drops a refused " +
			"plugin's units with no error. It changes nothing. Where each unit is there, it prints a line " +
			"that does not change.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			r, err := loadUnits(c, p, s)
			defer r.term.Flush()
			if err != nil {
				return err
			}
			text, err := p.Machine.Run(c.Context(), "mise", "bootstrap", "packages", "status", "--json")
			if errors.Is(err, exec.ErrNotFound) {
				return errors.New("mise is not on PATH")
			}
			if err != nil {
				return fmt.Errorf("reading mise's packages: %w", err)
			}
			missing, err := unplanned(text, kind.Names(r.all), r.runDir())
			if err != nil {
				return err
			}
			if len(missing) > 0 {
				return fmt.Errorf("mise's plan lacks %s: add each to [bootstrap.packages], or run mise in %s",
					strings.Join(missing, ", "), r.runDir())
			}
			_, err = fmt.Fprintln(c.OutOrStdout(), "ws units: none missing")
			return err
		},
	}
}

// unplanned returns the units, as mise names them, that mise's status lacks. A ws manager that mise
// did not load is an error, as each unit is then missing. It names runDir too, as a run outside it
// loads no line of ws's.
func unplanned(status []byte, units []string, runDir string) ([]string, error) {
	var st miseStatus
	if err := json.Unmarshal(status, &st); err != nil {
		return nil, fmt.Errorf("reading mise's packages: %w", err)
	}
	ws, ok := st[manager]
	if !ok || !ws.Available {
		return nil, fmt.Errorf("mise has no %s plugin loaded: link /opt/ws/plugins/%s, root's alone, where "+
			"mise finds a system plugin, as setup does, or run mise in %s", manager, manager, runDir)
	}
	planned := map[string]bool{}
	for _, pkg := range ws.Packages {
		planned[pkg.Package] = true
	}
	var missing []string
	for _, u := range units {
		if !planned[u] {
			missing = append(missing, manager+":"+u)
		}
	}
	return missing, nil
}
