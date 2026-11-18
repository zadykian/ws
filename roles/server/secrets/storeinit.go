package secrets

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
)

// Found says which of the credential and the file Init found. What it did not find, it made.
type Found struct{ Credential, File bool }

// Init makes a missing credential, and the file where none is. A file without the
// credential stops it and makes nothing, as a new identity would not open the file.
func (s Store) Init(ctx context.Context) (Found, error) {
	cred := credstore(s)
	has, err := cred.Has()
	if err != nil {
		return Found{}, err
	}
	_, err = s.M.Stat(FilePath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Found{}, err
	}
	found := Found{Credential: has, File: err == nil}
	if !has && found.File {
		return Found{}, fmt.Errorf("%s is there without the credential %s, and a new identity would not open it: "+
			"remove the file, then run init again", FilePath, CredstorePath)
	}
	if !has {
		if _, err := cred.Make(ctx); err != nil {
			return Found{}, err
		}
	}
	if found.File {
		return found, nil
	}
	// An empty edit reads the identity back from the credential, which proves that it opens.
	return found, s.Edit(ctx, func(*Secrets) error { return nil })
}
