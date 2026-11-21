package cli

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/signoz"
	"github.com/zadykian/ws/roles/server/values"
)

// newSignozCmd returns ws server signoz: SigNoz's steps that no kind holds, which the
// signoz role's run units call.
func newSignozCmd(p enginecli.Env) *cobra.Command {
	password := signozSteps(p, "password", "write SigNoz's root password to "+signoz.Dir+"/root.env, once",
		"It draws the password where root.env is missing, and keeps the one there, whatever it holds, "+
			"with root's owner and mode 600.",
		func(m effect.Machine, _ effect.Out) ([]step.Step, error) {
			return []step.Step{signoz.Password(m)}, nil
		})
	sync := signozSteps(p, "sync", "set SigNoz's retention and dashboards through its API",
		"It waits for SigNoz's API at WS_SIGNOZ_URL, then sets the retention of WS_SIGNOZ_RETENTION_DAYS, "+
			"and creates or replaces each dashboard of WS_SIGNOZ_DASHBOARDS. Without booted systemd it skips.",
		func(m effect.Machine, out effect.Out) ([]step.Step, error) {
			v, err := values.Load(m, serverVars)
			return signoz.Sync(v.Signoz, m, out), err
		})
	return enginecli.Group("signoz", "SigNoz's steps that the signoz role's units run", password, sync)
}

// signozSteps returns a command of ws server signoz that runs the steps that steps returns.
// --check plans them and applies none, and fails where one has a change to make: a run unit's
// check reads its exit status.
func signozSteps(p enginecli.Env, use, short, long string,
	steps func(m effect.Machine, out effect.Out) ([]step.Step, error),
) *cobra.Command {
	c := &cobra.Command{Use: use, Short: short, Long: long, Args: cobra.NoArgs}
	check := c.Flags().Bool("check", false, "plan the steps, apply none, and fail where one has a change to make")
	c.RunE = func(c *cobra.Command, _ []string) error {
		if p.Euid != 0 {
			return errors.New("run it as root")
		}
		term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
		defer term.Flush()
		out := linesOf(term, c.CommandPath())
		s, err := steps(p.Machine, out)
		if err != nil {
			return err
		}
		changed, err := step.Run(c.Context(), s, out, *check)
		if err == nil && *check && changed {
			err = errors.New("changes are due")
		}
		return err
	}
	return c
}
