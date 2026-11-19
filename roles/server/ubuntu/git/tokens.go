package git

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/roles/server/internal/plan"
)

// Tokens is the credential programs of git's tokens for servers besides GitHub, in
// /etc/ws/tokens.gitconfig, one per URL of TokensFile. Each names ws's program and the URL's key,
// never a token (roles/server/ubuntu/git/README.md#git-credentials). The file follows TokensFile
// whole: a URL the file lacks loses its program, and without the file, it goes. A run without the
// secrets env keeps TokensFile, so it keeps the programs too.
type Tokens struct {
	M effect.Machine
	A account.Account
}

// Plan reads TokensFile, which mise's files step wrote before any unit, and refuses it before any
// write.
func (t Tokens) Plan(ctx context.Context) ([]resource.Change, error) {
	tokens, _, err := ReadTokens(t.M)
	if err != nil {
		return nil, err
	}
	creds := credentialsOf(tokens)
	if len(creds) == 0 {
		return plan.All(ctx, file.Absent{M: t.M, Path: tokensPath})
	}
	content, err := tokensConfig(sections(creds))
	if err != nil {
		return nil, err
	}
	return plan.All(ctx, file.Validated{File: file.File{M: t.M, Path: tokensPath, Content: content,
		Mode: 0o640, GID: t.A.GID}, Check: configCheck})
}

// configCheck is the command that has git read a config file before it replaces the old one.
func configCheck(path string) (string, []string) {
	return "git", []string{"config", "--file", path, "--name-only", "--list"}
}
