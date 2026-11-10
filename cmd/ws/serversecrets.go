package main

import (
	"errors"
	"fmt"
	"io/fs"

	"filippo.io/age"
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/secrets"
)

//nolint:gosec // G101: the path of the encrypted file, no credential
const serverSecrets = "/etc/ws/secrets.age"

// newServerSecretsCmd returns ws server secrets, which keeps ws's secrets file, with its identity as
// a systemd credential. Its set, unset and keys are ws mac secrets'.
func newServerSecretsCmd(p process) *cobra.Command {
	open := func() (secretsStore, error) { return serverSecretsStore(p) }
	return group("secrets", "keep ws's secrets file, with its identity as a systemd credential",
		append([]*cobra.Command{{
			Use:   "init",
			Short: "make ws's age identity and an empty secrets file",
			Long: "It makes ws's age identity in memory, which systemd-creds encrypts by the host's key as the " +
				"credential " + secrets.CredstorePath + ", for a unit's LoadCredentialEncrypted=" +
				secrets.CredentialName + ". It makes an empty " + serverSecrets + ", root's and 600. A credential " +
				"or a file that is there stays.",
			Args: cobra.NoArgs,
			RunE: func(c *cobra.Command, _ []string) error { return serverSecretsInit(c, p) },
		}}, secretsEditCmds(p, open)...)...)
}

// serverSecretsStore returns ws's store, to root alone: the file and the host's key are root's.
func serverSecretsStore(p process) (secretsStore, error) {
	if p.euid != 0 {
		return secretsStore{}, errors.New("run it as root")
	}
	return secretsStore{m: p.machine, k: secrets.Credstore{M: p.machine}, path: serverSecrets,
		write: func(s secrets.Secrets, id age.Identity) error { return writeServerSecrets(p.machine, s, id) }}, nil
}

// writeServerSecrets replaces ws's file, root's and 600, beside the values.
func writeServerSecrets(m effect.Machine, s secrets.Secrets, id age.Identity) error {
	data, err := secrets.Seal(s, id)
	if err != nil {
		return err
	}
	if err := makeConfigDir(m); err != nil {
		return err
	}
	return m.WriteAtomic(serverSecrets, data, effect.Attrs{Mode: 0o600})
}

// serverSecretsInit makes a missing credential, and the file where none is. A file
// without the credential stops it, as a new identity would not open the file.
func serverSecretsInit(c *cobra.Command, p process) error {
	store, err := serverSecretsStore(p)
	if err != nil {
		return err
	}
	cred := secrets.Credstore{M: p.machine}
	has, err := cred.Has()
	if err != nil {
		return err
	}
	_, err = p.machine.Stat(serverSecrets)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	file := err == nil
	switch {
	case has:
		sayOut(c, "the credential %s is there, and stays", secrets.CredstorePath)
	case file:
		return fmt.Errorf("%s is there without the credential %s, and a new identity would not open it: "+
			"remove the file, then run init again", serverSecrets, secrets.CredstorePath)
	default:
		if _, err := cred.Make(c.Context()); err != nil {
			return err
		}
		sayOut(c, "made the identity, the credential %s", secrets.CredstorePath)
	}
	if file {
		sayOut(c, "%s is there, and stays", serverSecrets)
		return nil
	}
	// An empty edit reads the identity back from the credential, which proves that it opens.
	if err := store.edit(c.Context(), func(*secrets.Secrets) error { return nil }); err != nil {
		return err
	}
	sayOut(c, "made %s", serverSecrets)
	return nil
}
