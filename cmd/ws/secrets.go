package main

import (
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/internal/keychain"
	"github.com/zadykian/ws/internal/secrets"
)

//nolint:gosec // G101: the path of the encrypted file, no credential
const secretsFile = ".config/ws/secrets.age"

// newSecretsCmd returns ws mac secrets, which keeps the secrets file in memory and the identity in
// the Keychain.
func newSecretsCmd(p process) *cobra.Command {
	return group("secrets", "keep the secrets file, with its identity in the Keychain",
		append([]*cobra.Command{newSecretsInitCmd(p)},
			secretsEditCmds(p, func() (secretsStore, error) { return newSecretsStore(p), nil })...)...)
}

// newSecretsInitCmd returns ws mac secrets init.
func newSecretsInitCmd(p process) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "make ws's age identity and an empty secrets file",
		Long: "It makes ws's age identity in the Keychain, as the item " + keychain.Service + ", and an empty " +
			"~/" + secretsFile + ". An item or a file that is there stays.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error { return secretsInit(c, p) },
	}
}

// secretsInit makes the identity where the Keychain lacks it, and the file where none is.
func secretsInit(c *cobra.Command, p process) error {
	say := func(format string, a ...any) { sayOut(c, format, a...) }
	k := keychain.Keychain{M: p.machine, User: p.getenv("USER")}
	made, err := k.Init(c.Context())
	if err != nil {
		return err
	}
	if made {
		say("made the identity, the Keychain item %s", keychain.Service)
	} else {
		say("the Keychain item %s is there, and stays", keychain.Service)
	}
	path := filepath.Join(p.getenv("HOME"), secretsFile)
	if _, err := p.machine.ReadFile(path); err == nil {
		say("%s is there, and stays", path)
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	id, err := k.Identity(c.Context())
	if err != nil {
		return err
	}
	if err := secrets.WriteFile(p.machine, path, secrets.Secrets{}, id); err != nil {
		return err
	}
	say("made %s", path)
	return nil
}
