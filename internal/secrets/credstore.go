package secrets

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"filippo.io/age"

	"github.com/zadykian/ws/effect"
)

// CredstorePath is the identity's credential on ws, where a unit's LoadCredentialEncrypted=ws-secrets
// finds it.
const CredstorePath = "/etc/credstore.encrypted/" + CredentialName

// FilePath is ws's secrets file on ws.
const FilePath = "/etc/ws/secrets.age"

// ErrNoCredential is the error where ws lacks the credential.
var ErrNoCredential = errors.New("no credential " + CredstorePath + ": run ws server secrets init")

// credstore keeps the identity as the credential, which systemd-creds encrypts by the host's key in
// /var/lib/systemd. It does not use the TPM's key: a change of firmware or of Secure Boot can lock
// the TPM's out. The key reaches systemd-creds on its stdin and leaves it on its stdout, never in
// an argument.
type credstore struct{ M effect.Machine }

// Has reports whether the credential is there. It reads no secret.
func (c credstore) Has() (bool, error) {
	_, err := c.M.Stat(CredstorePath)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// Identity decrypts the identity from the credential. The error names no key.
func (c credstore) Identity(ctx context.Context) (age.Identity, error) {
	if has, err := c.Has(); err != nil || !has {
		return nil, errors.Join(err, ErrNoCredential)
	}
	out, err := c.M.Run(ctx, "systemd-creds", "decrypt", "--name="+CredentialName, CredstorePath, "-")
	if err != nil {
		return nil, fmt.Errorf("systemd-creds can't decrypt %s: %w", CredstorePath, err)
	}
	return ParseIdentity(string(out))
}

// Make makes an identity in memory and writes it as the credential, mode 600 in a directory of 700,
// replacing one that is there. ws writes the encrypted bytes itself, as it writes its other files.
func (c credstore) Make(ctx context.Context) (*age.X25519Identity, error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, fmt.Errorf("make the age identity: %w", err)
	}
	key := id.String()
	sealed, err := c.M.RunInput(ctx, []byte(key+"\n"), "systemd-creds", "encrypt", "--with-key=host",
		"--name="+CredentialName, "-", "-")
	if err != nil {
		return nil, fmt.Errorf("systemd-creds can't encrypt the identity: %w", scrubKey(err, key))
	}
	dir := path.Dir(CredstorePath)
	if err := c.M.Mkdir(dir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, err
	}
	// systemd's tmpfiles make it 700; one that ws finds otherwise is made so.
	if err := c.M.Chmod(dir, 0o700); err != nil {
		return nil, err
	}
	if err := c.M.WriteAtomic(CredstorePath, sealed, effect.Attrs{Mode: 0o600}); err != nil {
		return nil, err
	}
	return id, nil
}

// scrubKey drops the key from an error, in case a command echoed it.
func scrubKey(err error, key string) error {
	if !strings.Contains(err.Error(), key) {
		return err
	}
	return errors.New(strings.ReplaceAll(err.Error(), key, "[key]"))
}
