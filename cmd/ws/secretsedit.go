package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/keychain"
	"github.com/zadykian/ws/internal/secrets"
)

// secretsStore is the secrets file and the identity that opens it.
type secretsStore struct {
	m    effect.Machine
	k    keychain.Keychain
	path string
}

// edit reads the file with its references as written, applies fn and writes the file again. A file
// that is missing is an empty one.
func (s secretsStore) edit(ctx context.Context, fn func(*secrets.Secrets) error) error {
	id, err := s.k.Identity(ctx)
	if err != nil {
		return err
	}
	cur, err := secrets.LoadRaw(s.m, s.path, id)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := fn(&cur); err != nil {
		return err
	}
	return secrets.WriteFile(s.m, s.path, cur, id)
}

// secretsSet runs set: KIND NAME with the value read, git URL with the flags, or file NAME PATH.
func secretsSet(ctx context.Context, args []string, s secretsStore, stdin io.Reader, stderr io.Writer) error {
	if len(args) < 2 {
		return errUsage
	}
	kind, name, rest := args[0], args[1], args[2:]
	switch kind {
	case secrets.KindFile:
		if len(rest) != 1 {
			return errUsage
		}
		data, err := s.m.ReadFile(rest[0])
		if err != nil {
			return err
		}
		return s.edit(ctx, func(c *secrets.Secrets) error { return c.SetFile(name, data) })
	case secrets.KindGit:
		g, err := gitFlags(rest)
		if err != nil {
			return err
		}
		if g.Token, err = readValue(stdin, stderr, kind+" "+name); err != nil {
			return err
		}
		return s.edit(ctx, func(c *secrets.Secrets) error { c.SetGit(name, g); return nil })
	}
	if len(rest) != 0 {
		return errUsage
	}
	value, err := readValue(stdin, stderr, kind+" "+name)
	if err != nil {
		return err
	}
	return s.edit(ctx, func(c *secrets.Secrets) error { return c.Set(kind, name, value) })
}

// gitFlags reads --username NAME and --proactive-auth.
func gitFlags(args []string) (secrets.Git, error) {
	var g secrets.Git
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--proactive-auth":
			g.ProactiveAuth = true
		case args[i] == "--username" && i+1 < len(args):
			i++
			g.Username = args[i]
		default:
			return g, errUsage
		}
	}
	return g, nil
}

// readValue reads a secret: at a terminal, a line with no echo, asked on stderr; else all of stdin,
// less one trailing newline. The error names no value.
func readValue(stdin io.Reader, stderr io.Writer, what string) (string, error) {
	var value string
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		_, _ = io.WriteString(stderr, what+": ")
		line, err := term.ReadPassword(int(f.Fd()))
		_, _ = io.WriteString(stderr, "\n")
		if err != nil {
			return "", fmt.Errorf("read the value of %s: %w", what, err)
		}
		value = string(line)
	} else {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("read the value of %s: %w", what, err)
		}
		value = strings.TrimSuffix(string(data), "\n")
	}
	if value == "" {
		return "", fmt.Errorf("the value of %s is empty", what)
	}
	return value, nil
}

// secretsKeys prints each entry as KIND NAME, never a value.
func secretsKeys(ctx context.Context, s secretsStore, stdout io.Writer) error {
	id, err := s.k.Identity(ctx)
	if err != nil {
		return err
	}
	cur, err := secrets.LoadRaw(s.m, s.path, id)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	for _, k := range cur.Keys() {
		_, _ = io.WriteString(stdout, k+"\n")
	}
	return nil
}

func newSecretsStore(m effect.Machine, getenv func(string) string) secretsStore {
	return secretsStore{m: m, k: keychain.Keychain{M: m, User: getenv("USER")},
		path: filepath.Join(getenv("HOME"), secretsFile)}
}
