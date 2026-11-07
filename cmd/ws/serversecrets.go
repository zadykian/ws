package main

import (
	"errors"

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
		secretsEditCmds(p, open)...)
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
