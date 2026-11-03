// Package keychain keeps ws's age identity in the Mac's Keychain, as the item ws-secrets. The key
// reaches security on its stdin and leaves it on its stdout, never in an argument.
package keychain

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"filippo.io/age"

	"github.com/zadykian/ws/effect"
)

// Service is the name of the Keychain item.
const Service = "ws-secrets"

// ErrNoItem is the error where the Keychain lacks the item.
var ErrNoItem = errors.New("the Keychain lacks the item " + Service + ": run ws mac secrets init")

// account is what security takes as the account in a command on stdin without a quote.
var account = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// Keychain is the item of one user.
type Keychain struct {
	M    effect.Machine
	User string
}

// Has reports whether the item is there. It reads no secret.
func (k Keychain) Has(ctx context.Context) (bool, error) {
	_, err := k.M.Run(ctx, "security", "find-generic-password", "-s", Service)
	if err == nil {
		return true, nil
	}
	if missing(err) {
		return false, nil
	}
	return false, fmt.Errorf("security can't find the item %s: %w", Service, err)
}

// Identity reads the identity from the item. The error names no key.
func (k Keychain) Identity(ctx context.Context) (age.Identity, error) {
	out, err := k.M.Run(ctx, "security", "find-generic-password", "-s", Service, "-w")
	if err != nil {
		if missing(err) {
			return nil, ErrNoItem
		}
		return nil, fmt.Errorf("security can't read the item %s: %w", Service, err)
	}
	ids, err := age.ParseIdentities(strings.NewReader(string(out)))
	if err != nil || len(ids) != 1 {
		return nil, errors.New("the item " + Service + " holds no age identity")
	}
	return ids[0], nil
}

// Init makes an identity and adds it as the item, by security -i with the command on its stdin. It
// does nothing where the item is there, and returns false.
func (k Keychain) Init(ctx context.Context) (bool, error) {
	if !account.MatchString(k.User) {
		return false, errors.New("the user name is empty or holds a character that ws does not pass to security")
	}
	has, err := k.Has(ctx)
	if err != nil || has {
		return false, err
	}
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return false, fmt.Errorf("make the age identity: %w", err)
	}
	cmd := fmt.Sprintf("add-generic-password -s %s -a %s -w %s\n", Service, k.User, id.String())
	if err := k.M.PassInput(ctx, []byte(cmd), "security", "-i"); err != nil {
		return false, fmt.Errorf("security can't add the item %s: %w", Service, scrub(err, id.String()))
	}
	return true, nil
}

// missing reports security's error for an item that is not there.
func missing(err error) bool {
	return strings.Contains(err.Error(), "could not be found") || strings.Contains(err.Error(), "exit status 44")
}

// scrub drops the key from an error, in case a command echoed it.
func scrub(err error, key string) error {
	if !strings.Contains(err.Error(), key) {
		return err
	}
	return errors.New(strings.ReplaceAll(err.Error(), key, "[key]"))
}
