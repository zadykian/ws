// Package git is the git role's procedures, which ws server ubuntu git runs for the role's units.
// They are the check, the test of the include, the tokens and the clones. See
// roles/server/ubuntu/git/README.md.
package git

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/roles/server/secrets"
	"github.com/zadykian/ws/roles/server/values"
)

// Check refuses what the role's units would act on: a clone that checkClones refuses, and an
// entry of the secrets file's [git] table that checkCredentials refuses. It returns the notes of
// what in the account's files shadows ws's settings, which name settings and sections, never a
// value. ws server ubuntu git check runs it, through the role's guard, before the tokens and the
// clones.
func Check(ctx context.Context, m effect.Machine, a account.Account, v values.Values, s secrets.Secrets) ([]string, error) {
	if err := checkClones(v.Git.Clones); err != nil {
		return nil, err
	}
	creds := Credentials(s.Git)
	if len(creds) > 0 {
		keys, err := tokenKeys(ctx, m)
		if err != nil {
			return nil, err
		}
		if s.Whole {
			keys = nil
		}
		if err := checkCredentials(creds, keys); err != nil {
			return nil, err
		}
	}
	text, err := a.ReadHome(m, ".gitconfig")
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, err
	}
	// A misplaced include is the include unit's to refuse: the notes leave it out.
	holds, err := Included(ctx, m, "~/.gitconfig", text)
	var out misplaced
	if err != nil && !errors.As(err, &out) {
		return nil, err
	}
	return shadow{M: m, A: a, Creds: creds, Whole: s.Whole}.notes(ctx, text, err == nil && !holds)
}

// tokenKeys returns the keys of tokens.gitconfig, none where the file is missing.
func tokenKeys(ctx context.Context, m effect.Machine) ([]string, error) {
	old, err := m.ReadRootFile(tokensPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return gitNames(ctx, m, old)
}
