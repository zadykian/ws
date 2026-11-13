package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/setup"
)

const daemonLabel = "com.github.zadykian.ws-tunnel"

// newTunnelLoadedCmd returns ws mac tunnel loaded.
func newTunnelLoadedCmd(p process) *cobra.Command {
	return &cobra.Command{
		Use:   "loaded",
		Short: "succeed where launchd has the tunnel's daemon loaded",
		Long:  "Anyone may read launchd's system domain, so it needs no sudo.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			if _, err := p.machine.Run(c.Context(), "launchctl", "print", "system/"+daemonLabel); err != nil {
				return fmt.Errorf("launchd has not loaded the daemon %s", daemonLabel)
			}
			return nil
		},
	}
}

// newTunnelRoutesCmd returns ws mac tunnel routes.
func newTunnelRoutesCmd(p process) *cobra.Command {
	var check bool
	c := &cobra.Command{
		Use:   "routes NAME [CIDR...]",
		Short: "route the networks into the tunnel, or drop NAME",
		Long:  "It has the tunnel take the networks of NAME, or drop the file of NAME where it gets none.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runRoute(c, check, func(out effect.Out) step.Step {
				return &setup.Routes{Set: args[0], Networks: args[1:], M: p.machine, Out: out, Check: check, Alone: true}
			})
		},
	}
	c.Flags().BoolVar(&check, "check", false, "print each change and make none")
	return c
}

// newResolverCmd returns ws mac resolver.
func newResolverCmd(p process) *cobra.Command {
	var check bool
	c := &cobra.Command{
		Use:   "resolver DOMAIN [ADDRESS...]",
		Short: "send DOMAIN's names to the nameservers, or drop it",
		Long:  "It has macOS ask the nameservers for DOMAIN, or drop the file of DOMAIN where it gets none.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runRoute(c, check, func(out effect.Out) step.Step {
				return &setup.Resolver{Domain: args[0], Addresses: args[1:], M: p.machine, Out: out}
			})
		},
	}
	c.Flags().BoolVar(&check, "check", false, "print each change and make none")
	return c
}

// runRoute runs the step that build makes, alone, with the command's lines. Under check it prints
// each change and makes none.
func runRoute(c *cobra.Command, check bool, build func(effect.Out) step.Step) error {
	term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
	defer term.Flush()
	out := progOut{term, c.CommandPath(), map[string]bool{}}
	_, err := step.RunWith(c.Context(), []step.Step{build(out)}, step.Options{Observer: out, Check: check})
	return unwrapStep(err)
}
