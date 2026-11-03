// Package keychain keeps ws's age identity in the Mac's Keychain, as the item ws-secrets. The key
// reaches security on its stdin and leaves it on its stdout, never in an argument.
package keychain

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"filippo.io/age"

	"github.com/zadykian/ws/effect"
)

// Service is the name of the Keychain item.
const Service = "ws-secrets"

// ErrNoItem is the error where the Keychain lacks the item.
var ErrNoItem = errors.New("the Keychain lacks the item " + Service + ": run ws mac secrets init")

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

// missing reports security's error for an item that is not there.
func missing(err error) bool {
	return strings.Contains(err.Error(), "could not be found") || strings.Contains(err.Error(), "exit status 44")
}
