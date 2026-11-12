package secrets

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"

	"filippo.io/age"

	"github.com/zadykian/ws/effect"
)

// Store is ws's secrets file and the identity's credential that opens it. A missing file holds no
// entry. The file is root's and 600, in /etc/ws, which the first write makes if missing.
type Store struct{ M effect.Machine }

// Edit reads the file with its references as written, applies fn, and writes the file again. An
// error of fn or of the seal writes nothing.
func (s Store) Edit(ctx context.Context, fn func(*Secrets) error) error {
	cur, id, err := s.read(ctx)
	if err != nil {
		return err
	}
	if err := fn(&cur); err != nil {
		return err
	}
	return s.write(cur, id)
}

// read returns the file's entries as written, with the identity that opens it.
func (s Store) read(ctx context.Context) (Secrets, age.Identity, error) {
	id, err := Credstore(s).Identity(ctx)
	if err != nil {
		return Secrets{}, nil, err
	}
	cur, err := LoadRaw(s.M, FilePath, id)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Secrets{}, nil, err
	}
	return cur, id, nil
}

// write replaces the file with cur, sealed to id.
func (s Store) write(cur Secrets, id age.Identity) error {
	data, err := Seal(cur, id)
	if err != nil {
		return err
	}
	dir := path.Dir(FilePath)
	if _, err := s.M.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		if err := s.M.Mkdir(dir, 0o755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return s.M.WriteAtomic(FilePath, data, effect.Attrs{Mode: 0o600})
}

// Seal marshals s, checks that Parse takes the result (Decode, then Expand), and encrypts it to the
// identity's recipient, in armor. An error names no value.
func Seal(s Secrets, id age.Identity) ([]byte, error) {
	x, ok := id.(*age.X25519Identity)
	if !ok {
		return nil, errors.New("the identity is no X25519 identity")
	}
	plain, err := Marshal(s)
	if err != nil {
		return nil, errors.New("marshal the secrets: not valid")
	}
	d, err := Decode(plain)
	if err != nil {
		return nil, fmt.Errorf("the secrets are not valid: %w", err)
	}
	// Expand's error quotes a name out of a value, so the message here does not carry it.
	if _, err := Expand(d); err != nil {
		return nil, errors.New("the secrets are not valid: a value holds a reference that is missing or loops, or a $ that is no reference")
	}
	return Encrypt(plain, x.Recipient())
}
