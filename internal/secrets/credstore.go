package secrets

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"filippo.io/age"

	"github.com/zadykian/ws/effect"
)

// CredstorePath is the identity's credential on ws, where a unit's LoadCredentialEncrypted=ws-secrets
// finds it.
const CredstorePath = "/etc/credstore.encrypted/" + CredentialName

// ErrNoCredential is the error where ws lacks the credential.
var ErrNoCredential = errors.New("no credential " + CredstorePath + ": run ws server secrets init")

// Credstore keeps the identity as the credential, which systemd-creds encrypts by the host's key in
// /var/lib/systemd. The key leaves systemd-creds on its stdout, never in an argument.
type Credstore struct{ M effect.Machine }

// Has reports whether the credential is there. It reads no secret.
func (c Credstore) Has() (bool, error) {
	_, err := c.M.Stat(CredstorePath)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// Identity decrypts the identity from the credential. The error names no key.
func (c Credstore) Identity(ctx context.Context) (age.Identity, error) {
	if has, err := c.Has(); err != nil || !has {
		return nil, errors.Join(err, ErrNoCredential)
	}
	out, err := c.M.Run(ctx, "systemd-creds", "decrypt", "--name="+CredentialName, CredstorePath, "-")
	if err != nil {
		return nil, fmt.Errorf("systemd-creds can't decrypt %s: %w", CredstorePath, err)
	}
	return ParseIdentity(string(out))
}
