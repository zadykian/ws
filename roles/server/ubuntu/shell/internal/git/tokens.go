package git

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/account"
	"github.com/zadykian/ws/engine/core/host/file"
	"github.com/zadykian/ws/roles/server/ubuntu/shell/internal/plan"
)

// Tokens is git's tokens for servers besides GitHub. Each URL's in a file of its own, which the
// URL's helper in ~/.config/git/tokens.gitconfig reads (docs/shell.md#git-credentials). So no token
// goes in a template, or in the diff a dry run shows. Without entries, as in CI, ws leaves the
// tokens and their helpers alone, so that a run without the secrets file changes nothing. With the
// secrets file as the whole truth, a URL that the entries omit loses its token and its helper;
// without, it keeps both.
type Tokens struct {
	M     effect.Machine
	A     account.Account
	Creds []Credential
	// Whole removes the tokens and helpers of the URLs that Creds omit.
	Whole bool
}

// Plan reads the keys of tokens.gitconfig, as a URL the entries omit keeps its section there, or
// loses it under Whole. It checks the entries before any token is written.
func (t Tokens) Plan(ctx context.Context) ([]host.Change, error) {
	if len(t.Creds) == 0 && !t.Whole {
		return nil, nil
	}
	config := ".config/git/tokens.gitconfig"
	var keys []string
	old, err := t.A.ReadHome(t.M, config)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		if keys, err = gitNames(ctx, t.M, old); err != nil {
			return nil, err
		}
	}
	dropped := keptURLs(keys, t.Creds)
	if t.Whole {
		keys = nil
	}
	if len(t.Creds) == 0 && len(dropped) == 0 {
		return nil, nil
	}
	if err := checkCredentials(t.Creds, keys); err != nil {
		return nil, err
	}
	content, err := tokensConfig(sections(keys, t.Creds))
	if err != nil {
		return nil, err
	}
	const rel = ".config/git/tokens"
	rs := []host.Resource{t.A.Dir(t.M, rel, 0o700)}
	if t.Whole {
		for _, u := range dropped {
			rs = append(rs, t.A.Absent(t.M, rel+"/"+credentialFile(u)))
		}
	}
	for _, c := range t.Creds {
		username := c.Username
		if username == "" {
			username = "token"
		}
		f := t.A.File(t.M, rel+"/"+credentialFile(c.URL),
			[]byte("username="+username+"\npassword="+c.Token+"\n"), 0o600)
		f.Secret = true
		rs = append(rs, f)
	}
	rs = append(rs, file.Validated{File: t.A.File(t.M, ".config/git/tokens.gitconfig", content, 0o600), Check: configCheck})
	return plan.All(ctx, rs...)
}
