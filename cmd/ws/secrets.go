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

const secretsUsage = "usage: ws mac secrets init\n\n" +
	"  init  make ws's age identity in the Keychain, as the item ws-secrets, and an empty\n" +
	"        ~/.config/ws/secrets.age; an item there stays\n"

//nolint:gosec // G101: the path of the encrypted file, no credential
const secretsFile = ".config/ws/secrets.age"

// secretsCommand runs ws mac secrets, which keeps the secrets file in memory and the identity in
// the Keychain.
func secretsCommand(ctx context.Context, args []string, m effect.Machine, getenv func(string) string,
	stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, secretsUsage)
		return 2
	}
	if args[0] == "init" && len(args) == 1 {
		return secretsInit(ctx, m, getenv, stdout, stderr)
	}
	if args[0] == "init" {
		_, _ = io.WriteString(stderr, secretsUsage)
		return 2
	}
	_, _ = fmt.Fprintf(stderr, "ws mac secrets: unknown command %q\n\n%s", args[0], secretsUsage)
	return 2
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
