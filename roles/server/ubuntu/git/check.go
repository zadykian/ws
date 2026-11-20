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
	"github.com/zadykian/ws/roles/server/values"
)

// Check refuses what the role's units would act on: a clone that checkClones refuses, and tokens
// that RunTokens refuses, by lookup. It returns the notes of what in the account's files shadows
// ws's settings, which name settings and sections, never a value. ws server ubuntu git check runs
// it, through the role's guard, before the tokens and the clones.
func Check(ctx context.Context, m effect.Machine, a account.Account, v values.Values,
	lookup func(string) (string, bool)) ([]string, error) {
	if err := checkClones(v.Git.Clones); err != nil {
		return nil, err
	}
	tokens, _, err := RunTokens(m, lookup)
	if err != nil {
		return nil, err
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
	return shadow{M: m, A: a, Creds: credentialsOf(tokens)}.notes(ctx, text, err == nil && !holds)
}
