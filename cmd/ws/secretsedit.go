package main

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"path/filepath"

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
