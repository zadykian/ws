package secrets

import (
	"bytes"
	"fmt"
	"io"

	"filippo.io/age"
	"filippo.io/age/armor"
)

// Decrypt returns the plain text of an age file in armor, for the identity. The error names no
// content.
func Decrypt(armored []byte, id age.Identity) ([]byte, error) {
	r, err := age.Decrypt(armor.NewReader(bytes.NewReader(armored)), id)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	plain, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plain, nil
}
