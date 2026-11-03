package secrets

import (
	"fmt"
	"path/filepath"

	"filippo.io/age"

	"github.com/zadykian/ws/effect"
)

// Load reads and decrypts the secrets file and the profile files that it names. A relative name
// is relative to the secrets file. An error names a file, never a value.
func Load(m effect.Machine, path string, id age.Identity) (Secrets, error) {
	plain, err := decryptFile(m, path, id)
	if err != nil {
		return Secrets{}, err
	}
	s, err := Parse(plain)
	if err != nil {
		return Secrets{}, fmt.Errorf("%s: %w", path, err)
	}
	for _, f := range []struct {
		name string
		to   *[]byte
	}{{s.Ikeward.P12File, &s.Files.P12}, {s.Ikeward.SswanFile, &s.Files.Sswan}} {
		if f.name == "" {
			continue
		}
		if !filepath.IsAbs(f.name) {
			f.name = filepath.Join(filepath.Dir(path), f.name)
		}
		if *f.to, err = decryptFile(m, f.name, id); err != nil {
			return Secrets{}, err
		}
	}
	return s, nil
}

func decryptFile(m effect.Machine, path string, id age.Identity) ([]byte, error) {
	data, err := m.ReadFile(path)
	if err != nil {
		return nil, err
	}
	plain, err := Decrypt(data, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return plain, nil
}
