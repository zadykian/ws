package secrets

import (
	"fmt"

	"filippo.io/age"

	"github.com/zadykian/ws/effect"
)

// Load reads and decrypts the secrets file. An error names the file, never a value.
func Load(m effect.Machine, path string, id age.Identity) (Secrets, error) {
	data, err := m.ReadFile(path)
	if err != nil {
		return Secrets{}, err
	}
	plain, err := Decrypt(data, id)
	if err != nil {
		return Secrets{}, fmt.Errorf("%s: %w", path, err)
	}
	s, err := Parse(plain)
	if err != nil {
		return Secrets{}, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}
