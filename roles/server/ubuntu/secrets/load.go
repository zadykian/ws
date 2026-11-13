package secrets

import (
	"fmt"

	"filippo.io/age"

	"github.com/zadykian/ws/engine/core/effect"
)

// Load reads and decrypts the secrets file. An error names the file, never a value.
func Load(m effect.Machine, path string, id age.Identity) (Secrets, error) {
	return load(m, path, id, Parse)
}

// loadRaw is Load with the references left as written, for an edit that writes the file again.
func loadRaw(m effect.Machine, path string, id age.Identity) (Secrets, error) {
	return load(m, path, id, Decode)
}

func load(m effect.Machine, path string, id age.Identity, parse func([]byte) (Secrets, error)) (Secrets, error) {
	data, err := m.ReadFile(path)
	if err != nil {
		return Secrets{}, err
	}
	plain, err := Decrypt(data, id)
	if err != nil {
		return Secrets{}, fmt.Errorf("%s: %w", path, err)
	}
	s, err := parse(plain)
	if err != nil {
		return Secrets{}, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}
