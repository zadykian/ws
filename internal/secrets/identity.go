package secrets

import (
	"errors"
	"fmt"
	"strings"

	"filippo.io/age"

	"github.com/zadykian/ws/effect"
)

// CredentialName is the name of the identity's systemd credential: the unit loads it by
// LoadCredential=CredentialName:PATH, and ws reads it from $CREDENTIALS_DIRECTORY.
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

// ReadCredential reads the identity from the credential CredentialName, in the directory that
// getenv gives as CREDENTIALS_DIRECTORY. Without the directory the error names LoadCredential.
func ReadCredential(m effect.Machine, getenv func(string) string) (age.Identity, error) {
	dir := getenv("CREDENTIALS_DIRECTORY")
	if dir == "" {
		return nil, errors.New("no credential " + CredentialName + ": run it as a unit with LoadCredential=" +
			CredentialName + ":PATH to the identity")
	}
	data, err := m.ReadFile(dir + "/" + CredentialName)
	if err != nil {
		return nil, fmt.Errorf("read the credential %s: %w", CredentialName, err)
	}
	return ParseIdentity(string(data))
}
