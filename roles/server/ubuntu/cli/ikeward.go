package cli

import (
	"errors"
	"io/fs"
	"strings"
	"time"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/roles/server/ubuntu/ikeward"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// newIkewardCmd returns ws server ubuntu ikeward: the commands that the ikeward role's units run.
// The wait is tries tries, pause apart.
func newIkewardCmd(p enginecli.Env, tries int, pause time.Duration) *cobra.Command {
	return enginecli.Group("ikeward", "the ikeward role's check, install and wait, which its units run",
		newIkewardCheckCmd(p), newIkewardInstallCmd(p), newIkewardWaitCmd(p, tries, pause))
}

// newIkewardCheckCmd returns ws server ubuntu ikeward check.
func newIkewardCheckCmd(p enginecli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "fail where the secrets hold no profile for WS_IKEWARD_PROFILE",
		Long: "It reads the values of " + serverVars + " and ws's secrets file, and fails where WS_IKEWARD_PROFILE " +
			"is empty or the secrets hold no profile for it, as install would. The ikeward role's guard runs it, " +
			"so that the run stops before ikeward's binary downloads, and a plan shows why.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			_, err := ikewardReady(c, p, nil)
			return err
		},
	}
}

// ikewardReady returns the install of the values' profile from the secrets, which says on out,
// or fails where the secrets hold no profile for it. A check, which applies nothing, passes no out.
func ikewardReady(c *cobra.Command, p enginecli.Env, out effect.Out) (ikeward.Install, error) {
	v, err := values.Load(p.Machine, serverVars)
	if err != nil {
		return ikeward.Install{}, err
	}
	if v.Ikeward.Profile == "" {
		return ikeward.Install{}, errors.New("WS_IKEWARD_PROFILE is empty, so there is no profile to install")
	}
	s, err := secrets.Store{M: p.Machine}.Load(c.Context())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return ikeward.Install{}, err
	}
	i := ikeward.Install{M: p.Machine, Out: out, Values: v.Ikeward, Secrets: s}
	return i, i.Ready()
}

// newIkewardInstallCmd returns ws server ubuntu ikeward install.
func newIkewardInstallCmd(p enginecli.Env) *cobra.Command {
	c := &cobra.Command{
		Use:   "install",
		Short: "run ikeward install from the profile of the secrets, where its dry run says it would change anything",
		Long: "It reads the values of " + serverVars + " and ws's secrets file. It writes the profile's two files into " +
			"/run/ikeward-profile, which only root may enter, runs ikeward install --dry-run on them with the .p12's " +
			"password on standard input, and removes them. Where the dry run exits 4, install would change " +
			"something, and it runs install. It fails where WS_IKEWARD_PROFILE is set and the secrets hold no " +
			"profile. Without booted systemd nothing is due.\n\n--check fails where install would change anything, " +
			"and changes nothing. The ikeward role's run state runs it, after ikeward's binary.",
		Args: cobra.NoArgs,
	}
	check := c.Flags().Bool("check", false, "fail where install would change anything, and change nothing")
	c.RunE = func(c *cobra.Command, _ []string) error {
		term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
		defer term.Flush()
		lines := linesOf(term, c.CommandPath())
		i, err := ikewardReady(c, p, lines)
		if err != nil {
			return err
		}
		return settle(c.Context(), lines, []resource.Resource{i}, *check)
	}
	return c
}

// newIkewardWaitCmd returns ws server ubuntu ikeward wait.
func newIkewardWaitCmd(p enginecli.Env, tries int, pause time.Duration) *cobra.Command {
	return &cobra.Command{
		Use:   "wait",
		Short: "wait for ikeward status to pass, as the VPN comes up",
		Long: "It runs /usr/local/sbin/ikeward status, and again until it passes, up to 12 times, 10 s apart, then " +
			"fails. A later unit may need the VPN, as for a clone from the remote network. Without booted systemd " +
			"nothing is due. The ikeward role's run state runs it where ikeward status fails, after the install.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
			defer term.Flush()
			lines := linesOf(term, c.CommandPath())
			w := ikeward.Status{M: p.Machine, Out: lines, Tries: tries, Wait: pause}
			return settle(c.Context(), lines, []resource.Resource{w}, false)
		},
	}
}
