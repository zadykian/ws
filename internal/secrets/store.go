package secrets

import (
	"errors"
	"fmt"

	"filippo.io/age"
)

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
