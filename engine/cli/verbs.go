package cli

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
)

// Apply returns the RunE of a side's apply command: it applies the Target in the process e.
func Apply(t engine.Target, e Env) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, _ []string) error {
		return engine.Apply(c.Context(), CallOf(c, e), t)
	}
}

// State returns the RunE of a side's state command: it reports the Target in the process e.
func State(t engine.Target, e Env) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, _ []string) error {
		return engine.State(c.Context(), CallOf(c, e), t)
	}
}
