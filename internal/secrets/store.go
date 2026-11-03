package secrets

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"filippo.io/age"

	"github.com/zadykian/ws/effect"
)

// WriteFile marshals s, checks that Parse takes the result (Decode, then Expand), encrypts it to the identity's
// recipient and replaces the file at path through a pending file, mode 0600. It makes the
// directories above path, 0700, where they are missing. An error names the path, never a value.
func WriteFile(m effect.Machine, path string, s Secrets, id age.Identity) error {
	x, ok := id.(*age.X25519Identity)
	if !ok {
		return errors.New("the identity is no X25519 identity")
	}
	plain, err := Marshal(s)
	if err != nil {
		return errors.New("marshal the secrets: not valid")
	}
	d, err := Decode(plain)
	if err != nil {
		return fmt.Errorf("the secrets are not valid: %w", err)
	}
	// Expand's error quotes a name out of a value, so the message here does not carry it.
	if _, err := Expand(d); err != nil {
		return errors.New("the secrets are not valid: a value holds a reference that is missing or loops, or a $ that is no reference")
	}
	armored, err := Encrypt(plain, x.Recipient())
	if err != nil {
		return err
	}
	for _, dir := range []string{filepath.Dir(filepath.Dir(path)), filepath.Dir(path)} {
		if err := m.Mkdir(dir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("%s: %w", dir, err)
		}
	}
	// A directory that exists keeps its mode from whoever made it, such as 755 under umask 022.
	if err := m.Chmod(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("%s: %w", filepath.Dir(path), err)
	}
	a := effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o600}
	if err := m.WriteAtomic(path, armored, a); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
