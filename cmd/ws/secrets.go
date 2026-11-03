package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/keychain"
	"github.com/zadykian/ws/internal/secrets"
)

var errUsage = errors.New("usage")

const secretsUsage = "usage: ws mac secrets COMMAND\n\n" +
	"  init  make ws's age identity in the Keychain, as the item ws-secrets, and an empty\n" +
	"        ~/.config/ws/secrets.age; an item there stays\n" +
	"  set token|env|password NAME  set the value, from a prompt with no echo or from stdin\n" +
	"  set git URL [--username NAME] [--proactive-auth]  set a git server's token, read as above\n" +
	"  set file NAME PATH  set a profile file (vpn_p12, vpn_sswan) from the file at PATH\n" +
	"  unset KIND NAME  drop an entry\n" +
	"  keys  print each entry as KIND NAME, never a value\n"

//nolint:gosec // G101: the path of the encrypted file, no credential
const secretsFile = ".config/ws/secrets.age"

// secretsCommand runs ws mac secrets, which keeps the secrets file in memory and the identity in
// the Keychain.
func secretsCommand(ctx context.Context, args []string, m effect.Machine, getenv func(string) string,
	stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, secretsUsage)
		return 2
	}
	store := newSecretsStore(m, getenv)
	var err error
	switch {
	case args[0] == "init" && len(args) == 1:
		return secretsInit(ctx, m, getenv, stdout, stderr)
	case args[0] == "set":
		err = secretsSet(ctx, args[1:], store, stdin, stderr)
	case args[0] == "unset" && len(args) == 3:
		err = store.edit(ctx, func(c *secrets.Secrets) error { return c.Unset(args[1], args[2]) })
	case args[0] == "keys" && len(args) == 1:
		err = secretsKeys(ctx, store, stdout)
	default:
		err = errUsage
	}
	if errors.Is(err, errUsage) {
		_, _ = io.WriteString(stderr, secretsUsage)
		return 2
	}
	if err != nil {
		return secretsFail(stderr, err)
	}
	return 0
}

// secretsInit makes the identity where the Keychain lacks it, and the file where none is.
func secretsInit(ctx context.Context, m effect.Machine, getenv func(string) string, stdout, stderr io.Writer) int {
	k := keychain.Keychain{M: m, User: getenv("USER")}
	made, err := k.Init(ctx)
	if err != nil {
		return secretsFail(stderr, err)
	}
	if made {
		_, _ = io.WriteString(stdout, "ws mac secrets: made the identity, the Keychain item "+keychain.Service+"\n")
	} else {
		_, _ = io.WriteString(stdout, "ws mac secrets: the Keychain item "+keychain.Service+" is there, and stays\n")
	}
	path := filepath.Join(getenv("HOME"), secretsFile)
	if _, err := m.ReadFile(path); err == nil {
		_, _ = io.WriteString(stdout, "ws mac secrets: "+path+" is there, and stays\n")
		return 0
	} else if !errors.Is(err, fs.ErrNotExist) {
		return secretsFail(stderr, err)
	}
	id, err := k.Identity(ctx)
	if err != nil {
		return secretsFail(stderr, err)
	}
	if err := secrets.WriteFile(m, path, secrets.Secrets{}, id); err != nil {
		return secretsFail(stderr, err)
	}
	_, _ = io.WriteString(stdout, "ws mac secrets: made "+path+"\n")
	return 0
}

func secretsFail(stderr io.Writer, err error) int {
	_, _ = fmt.Fprintf(stderr, "ws mac secrets: %v\n", err)
	return 1
}
