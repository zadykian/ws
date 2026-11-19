package secrets

import (
	"errors"
	"fmt"
	"strings"

	"filippo.io/age"
)

// CredentialName is the name of the identity's systemd credential: systemd-creds names it so, and
// ws unit decrypts it by systemd-creds decrypt.
const CredentialName = "ws-secrets"

// ParseIdentity parses text as one X25519 identity. Comment lines are skipped. The error names no
// content.
func ParseIdentity(text string) (age.Identity, error) {
	ids, err := age.ParseIdentities(strings.NewReader(text))
	if err != nil {
		return nil, errors.New("the age identity is not valid")
	}
	if len(ids) != 1 {
		return nil, fmt.Errorf("want one age identity, got %d", len(ids))
	}
	return ids[0], nil
}
