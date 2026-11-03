package secrets

import (
	"bytes"
	"fmt"

	"filippo.io/age"
	"filippo.io/age/armor"
)

// Encrypt returns plain as an age file in armor, for the recipient. The error names no content.
func Encrypt(plain []byte, r age.Recipient) ([]byte, error) {
	var buf bytes.Buffer
	a := armor.NewWriter(&buf)
	w, err := age.Encrypt(a, r)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}
	if _, err := w.Write(plain); err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}
	if err := a.Close(); err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}
	return buf.Bytes(), nil
}
