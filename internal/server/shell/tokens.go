package shell

import (
	"context"
	"errors"
	"io/fs"
	"path"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/values"
)

// tokens is git's tokens for servers besides GitHub, each URL's in a file of its own, which the
// URL's helper in ~/.config/git/tokens.gitconfig reads (docs/shell.md#git-credentials). So no token
// goes in a template, or in the diff a dry run shows. Without entries, as in CI, ws leaves the
// tokens and their helpers alone, so that a run without the secrets file changes nothing. Only an
// entry with state absent removes a token or its helper.
type tokens struct {
	M     effect.Machine
	A     account
	Creds []values.GitCredential
}

// Plan reads the keys of tokens.gitconfig, as a URL the entries omit keeps its section there, and
// checks the entries before any token is written.
func (t tokens) Plan(ctx context.Context) ([]host.Change, error) {
	if len(t.Creds) == 0 {
		return nil, nil
	}
	config := t.A.Home + "/.config/git/tokens.gitconfig"
	var keys []string
	old, err := t.M.ReadRootFile(config)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		if keys, err = gitNames(ctx, t.M, old); err != nil {
			return nil, err
		}
	}
	if err := checkCredentials(t.Creds, keys); err != nil {
		return nil, err
	}
	content, err := tokensConfig(sections(keys, t.Creds))
	if err != nil {
		return nil, err
	}
	dir := t.A.Home + "/.config/git/tokens"
	rs := treeTo(t.M, t.A, path.Dir(config), 0o755, true)
	rs = append(rs, host.Dir{M: t.M, Path: dir, Mode: 0o700, UID: t.A.UID, GID: t.A.GID})
	for _, c := range t.Creds {
		file := dir + "/" + credentialFile(c.URL)
		if c.State == "absent" {
			rs = append(rs, host.Absent{M: t.M, Path: file})
			continue
		}
		username := c.Username
		if username == "" {
			username = "token"
		}
		rs = append(rs, host.File{
			M: t.M, Path: file, Content: []byte("username=" + username + "\npassword=" + c.Token + "\n"),
			Mode: 0o600, UID: t.A.UID, GID: t.A.GID, Secret: true,
		})
	}
	rs = append(rs, host.Validated{
		File:  host.File{M: t.M, Path: config, Content: content, Mode: 0o600, UID: t.A.UID, GID: t.A.GID},
		Check: configCheck,
	})
	return planAll(ctx, rs...)
}
