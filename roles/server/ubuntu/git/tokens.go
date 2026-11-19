package git

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/roles/server/internal/plan"
)

// Tokens is git's tokens for servers besides GitHub. Each URL's in a file of its own, which the
// URL's credential program in /etc/ws/tokens.gitconfig reads
// (roles/server/ubuntu/git/README.md#git-credentials). So no token
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
func (t Tokens) Plan(ctx context.Context) ([]resource.Change, error) {
	if len(t.Creds) == 0 && !t.Whole {
		return nil, nil
	}
	var keys []string
	old, err := t.M.ReadRootFile(tokensPath)
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
	rs := []resource.Resource{file.Dir{M: t.M, Path: tokensDir, Mode: 0o750, GID: t.A.GID}}
	if t.Whole {
		for _, u := range dropped {
			rs = append(rs, file.Absent{M: t.M, Path: tokensDir + "/" + credentialKey(u)})
		}
	}
	for _, c := range t.Creds {
		username := c.Username
		if username == "" {
			username = "token"
		}
		rs = append(rs, file.File{M: t.M, Path: tokensDir + "/" + credentialKey(c.URL),
			Content: []byte("username=" + username + "\npassword=" + c.Token + "\n"),
			Mode:    0o640, GID: t.A.GID, Secret: true})
	}
	rs = append(rs, file.Validated{File: file.File{M: t.M, Path: tokensPath, Content: content,
		Mode: 0o640, GID: t.A.GID}, Check: configCheck})
	return plan.All(ctx, rs...)
}

// configCheck is the command that has git read a config file before it replaces the old one.
func configCheck(path string) (string, []string) {
	return "git", []string{"config", "--file", path, "--name-only", "--list"}
}
