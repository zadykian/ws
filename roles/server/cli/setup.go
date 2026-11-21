package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/fetch"
	"github.com/zadykian/ws/roles/internal/setup"
)

// newSetupCmd returns ws server setup, which sets the server up as root: it checks the system and
// installs mise at the release that mise pins.
func newSetupCmd(p enginecli.Env, mise setup.MiseRelease) *cobra.Command {
	c := &cobra.Command{
		Use:   "setup",
		Short: "set the server up, as root",
		Long: "It refuses a user but root, and a system but Ubuntu 26.04, before any write. It installs mise " +
			"at the version ws pins, checked against its SHA-256, where mise --version differs, and writes " +
			setup.MisercPath + " and " + setup.ConfigPath + ". A second run writes nothing.",
		Args: cobra.NoArgs,
	}
	c.RunE = func(c *cobra.Command, _ []string) error {
		if p.Euid != 0 {
			return errors.New("run it as root")
		}
		if err := ubuntu(p.Machine); err != nil {
			return err
		}
		steps, err := setupSteps(c.Context(), p.Machine, mise)
		if err != nil {
			return err
		}
		term := effect.NewTerminal(c.ErrOrStderr())
		defer term.Flush()
		return settle(c.Context(), linesOf(term, c.CommandPath()), steps, false)
	}
	return c
}

// setupSteps returns the resources of setup, in order. Each one plans before it writes.
func setupSteps(ctx context.Context, m effect.Machine, mise setup.MiseRelease) ([]resource.Resource, error) {
	target, err := miseTarget(ctx, m)
	if err != nil {
		return nil, err
	}
	return append([]resource.Resource{setup.MiseBinary{M: m, Release: mise, Target: target}},
		setup.SystemMise(m, "server")...), nil
}

// miseTarget names mise's binary for this machine by dpkg's architecture: ws sets up x86_64 and
// aarch64 alone.
func miseTarget(ctx context.Context, m effect.Machine) (string, error) {
	arch, err := fetch.Architecture(ctx, m)
	if err != nil {
		return "", err
	}
	switch arch {
	case "amd64":
		return "linux-x64", nil
	case "arm64":
		return "linux-arm64", nil
	}
	return "", fmt.Errorf("ws sets up Linux on x86_64 and aarch64, not on %s", arch)
}
