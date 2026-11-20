package git

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/file"
)

// Tokens is the credential programs of git's tokens for servers besides GitHub, in
// /etc/ws/tokens.gitconfig, one per URL of TokensFile. Each names ws's program and the URL's key,
// never a token (roles/server/ubuntu/git/README.md#git-credentials). The file follows TokensFile
// whole: a URL the file lacks loses its program, and without the file, it goes. A run without the
// secrets env keeps TokensFile, so it keeps the programs too.
type Tokens struct {
	M effect.Machine
	A account.Account
	// Lookup gives the secrets task's input, which RunTokens takes over TokensFile. Nil reads the
	// file.
	Lookup func(string) (string, bool)
}

// Plan reads the tokens as RunTokens gives them, and refuses them before any write.
func (t Tokens) Plan(ctx context.Context) ([]resource.Change, error) {
	tokens, _, err := RunTokens(t.M, t.Lookup)
	if err != nil {
		return nil, err
	}
	creds := credentialsOf(tokens)
	if len(creds) == 0 {
		return file.Absent{M: t.M, Path: tokensPath}.Plan(ctx)
	}
	content, err := tokensConfig(sections(creds))
	if err != nil {
		return nil, err
	}
	return file.Validated{File: file.File{M: t.M, Path: tokensPath, Content: content,
		Mode: 0o640, GID: t.A.GID}, Check: configCheck}.Plan(ctx)
}

// configCheck is the command that has git read a config file before it replaces the old one.
func configCheck(path string) (string, []string) {
	return "git", []string{"config", "--file", path, "--name-only", "--list"}
}
