package cli

import (
	"errors"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/roles/client/macos/secrets"
)

// newSecretsCmd returns ws client macos secrets, the commands of the secrets repository.
func newSecretsCmd(p enginecli.Env, server *string) *cobra.Command {
	return enginecli.Group("secrets", "the commands of the secrets repository, which fnox keeps",
		newSecretsInitCmd(p, server), newSecretsAddCmd(p, server))
}

// newSecretsInitCmd returns ws client macos secrets init.
func newSecretsInitCmd(p enginecli.Env, server *string) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "set the secrets repository up with the Nano and a backup key",
		Long: "It reads the repository from WS_SECRETS_REPO in /opt/ws/mise.toml, and clones it. It takes the age " +
			"identity that the YubiKey Nano holds, or makes one, which asks for its PIN and a touch, and keeps its " +
			"stub in fnox's key file. Where fnox.toml is missing, it makes a backup key, starts fnox.toml with both " +
			"recipients, and prints the backup's secret line once. It declares each input of /opt/ws's server " +
			"roles that fnox.toml lacks, empty. It lands the change by a pull request once its checks pass, and " +
			"opens none where nothing changed. It stops where a pull request of an earlier run is still open. Run " +
			"it again once /opt/ws holds a release that declares a new input.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			if p.Euid == 0 {
				return errors.New("run it as yourself, not root: the Nano, fnox's key file and gh's login are yours")
			}
			if *server != "" {
				return errors.New("init takes no --server: the secrets repository serves every server")
			}
			return secrets.Init{M: p.Machine, Getenv: p.Getenv, Out: c.OutOrStdout(), In: c.InOrStdin(),
				Tree: secrets.Tree}.Run(c.Context())
		},
	}
}
