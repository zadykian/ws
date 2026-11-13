package cli

import (
	"errors"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

// NewSecretsCmd returns ws server ubuntu secrets, which keeps ws's secrets file, with its
// identity as a systemd credential. ws client macos secrets runs its commands here, over ssh.
func NewSecretsCmd(p enginecli.Env) *cobra.Command {
	open := func() (secrets.Store, error) { return ServerSecretsStore(p) }
	return enginecli.Group("secrets", "keep ws's secrets file, with its identity as a systemd credential",
		append([]*cobra.Command{{
			Use:   "init",
			Short: "make ws's age identity and an empty secrets file",
			Long: "It makes ws's age identity in memory, which systemd-creds encrypts by the host's key as the " +
				"credential " + secrets.CredstorePath + ", for a unit's LoadCredentialEncrypted=" +
				secrets.CredentialName + ". It makes an empty " + secrets.FilePath + ", root's and 600. A credential " +
				"or a file that is there stays.",
			Args: cobra.NoArgs,
			RunE: func(c *cobra.Command, _ []string) error { return serverSecretsInit(c, p) },
		}}, SecretsEdit(p, open)...)...)
}

// ServerSecretsStore returns ws's store, to root alone: the file and the host's key are root's.
func ServerSecretsStore(p enginecli.Env) (secrets.Store, error) {
	if p.Euid != 0 {
		return secrets.Store{}, errors.New("run it as root")
	}
	return secrets.Store{M: p.Machine}, nil
}

// serverSecretsInit makes a missing credential, and the file where none is, and says which it made.
func serverSecretsInit(c *cobra.Command, p enginecli.Env) error {
	store, err := ServerSecretsStore(p)
	if err != nil {
		return err
	}
	found, err := store.Init(c.Context())
	if err != nil {
		return err
	}
	if found.Credential {
		sayOut(c, "the credential %s is there, and stays", secrets.CredstorePath)
	} else {
		sayOut(c, "made the identity, the credential %s", secrets.CredstorePath)
	}
	if found.File {
		sayOut(c, "%s is there, and stays", secrets.FilePath)
	} else {
		sayOut(c, "made %s", secrets.FilePath)
	}
	return nil
}
