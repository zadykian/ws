package cli

import (
	"errors"
	"time"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/roles/server/ubuntu/ikeward"
	"github.com/zadykian/ws/roles/server/values"
)

// newIkewardCmd returns ws server ikeward: the commands that the ikeward role's units run.
// The wait is tries tries, pause apart.
func newIkewardCmd(p enginecli.Env, tries int, pause time.Duration) *cobra.Command {
	return enginecli.Group("ikeward", "the ikeward role's check, install and wait, which its units run",
		newIkewardCheckCmd(p), newIkewardInstallCmd(p), newIkewardWaitCmd(p, tries, pause))
}

// newIkewardCheckCmd returns ws server ikeward check.
func newIkewardCheckCmd(p enginecli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "fail where the run's secrets hold no profile for WS_IKEWARD_PROFILE",
		Long: "It reads the values of " + serverVars + " and the profile's variables, " + ikeward.P12Var + ", " +
			ikeward.SswanVar + " and " + ikeward.PasswordVar + ", which the secrets task sets. It fails where " +
			"WS_IKEWARD_PROFILE is empty, or where the variables are set and either file's is empty, as install " +
			"would. A run without them passes. The ikeward role's guard runs it, so that the run stops before " +
			"ikeward's binary downloads, and a plan shows why.",
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			_, err := ikewardReady(p, nil)
			return err
		},
	}
}

// ikewardReady returns the install of the values' profile from the run's env, which says on out,
// or fails where the run's secrets hold no profile for it. A check, which applies nothing, passes
// no out.
func ikewardReady(p enginecli.Env, out effect.Out) (ikeward.Install, error) {
	v, err := values.Load(p.Machine, serverVars)
	if err != nil {
		return ikeward.Install{}, err
	}
	if v.Ikeward.Profile == "" {
		return ikeward.Install{}, errors.New("WS_IKEWARD_PROFILE is empty, so there is no profile to install")
	}
	profile, inRun, err := ikeward.ProfileFrom(p.LookupEnv)
	if err != nil {
		return ikeward.Install{}, err
	}
	i := ikeward.Install{M: p.Machine, Out: out, Values: v.Ikeward, Profile: profile, InRun: inRun}
	return i, i.Ready()
}

// newIkewardInstallCmd returns ws server ikeward install.
func newIkewardInstallCmd(p enginecli.Env) *cobra.Command {
	c := &cobra.Command{
		Use:   "install",
		Short: "run ikeward install from the profile of the run's secrets, where its dry run says it would change anything",
		Long: "It reads the values of " + serverVars + " and the profile's variables, as check does. It writes the " +
			"profile's two files into /run/ikeward-profile, which only root may enter, runs ikeward install " +
			"--dry-run on them with the .p12's password on standard input, and removes them. Where the dry run " +
			"exits 4, install would change something, and it runs install. It fails where check would. Without " +
			"the variables, as in a run without the secrets, nothing is due.\n\n" +
			"--check fails where install would change anything, and changes nothing. The ikeward role's run state " +
			"runs it, after ikeward's binary.",
		Args: cobra.NoArgs,
	}
	check := c.Flags().Bool("check", false, "fail where install would change anything, and change nothing")
	c.RunE = func(c *cobra.Command, _ []string) error {
		term := effect.NewTerminal(c.ErrOrStderr())
		defer term.Flush()
		lines := linesOf(term, c.CommandPath())
		i, err := ikewardReady(p, lines)
		if err != nil {
			return err
		}
		return settle(c.Context(), lines, []resource.Resource{i}, *check)
	}
	return c
}

// newIkewardWaitCmd returns ws server ikeward wait.
func newIkewardWaitCmd(p enginecli.Env, tries int, pause time.Duration) *cobra.Command {
	c := &cobra.Command{
		Use:   "wait",
		Short: "wait for ikeward status to pass, as the VPN comes up",
		Long: "It runs /usr/local/sbin/ikeward status, and again until it passes, up to 12 times, 10 s apart, then " +
			"fails. A later unit may need the VPN, as for a clone from the remote network. Without the profile's " +
			"variables, as in a run without the secrets, which installs no VPN, nothing is due.\n\n--check fails where it would wait, and changes nothing. The ikeward role's run state runs " +
			"it, after the install.",
		Args: cobra.NoArgs,
	}
	check := c.Flags().Bool("check", false, "fail where the VPN is down and the run would wait, and change nothing")
	c.RunE = func(c *cobra.Command, _ []string) error {
		term := effect.NewTerminal(c.ErrOrStderr())
		defer term.Flush()
		lines := linesOf(term, c.CommandPath())
		w := ikeward.Status{M: p.Machine, Out: lines, Tries: tries, Wait: pause, InRun: ikeward.InRun(p.LookupEnv)}
		return settle(c.Context(), lines, []resource.Resource{w}, *check)
	}
	return c
}
