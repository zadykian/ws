// Package git is the git role as Go steps: ~/.gitconfig, the key that signs commits, the allowed
// signers, the tokens for servers besides GitHub and the clones. See roles/server/ubuntu/git/README.md.
package git

import (
	"context"
	"embed"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/roles/internal/etcws"
	"github.com/zadykian/ws/roles/internal/gitcfg"
	"github.com/zadykian/ws/roles/server/ubuntu/internal/plan"
)

// gitFiles are the templates of /etc/ws/gitconfig and of the tokens' include.
//
//go:embed files/gitconfig.tmpl files/tokens.gitconfig.tmpl
var gitFiles embed.FS

// Config is ws's git settings in /etc/ws/gitconfig, root's and 644, and the include of it at the
// top of ~/.gitconfig, which stays the user's. A real run keeps a ~/.gitconfig that lacks the
// include beside it, as ~/.gitconfig.*~. The user's file has no diff, as it may hold a credential,
// such as an http.extraHeader, that ws's lines never show.
type Config struct {
	M   effect.Machine
	Out effect.Out
	A   account.Account
	Key string
	// Creds are the tokens of this run, and Whole says that the run holds every token: the role
	// then ignores the URLs that tokens.gitconfig still holds.
	Creds []Credential
	Whole bool
}

func (g Config) content() ([]byte, error) {
	settings := gitcfg.Settings(gitcfg.Inputs{Key: g.Key, Gh: "/usr/bin/gh"})
	return resource.Render(gitFiles, "files/gitconfig.tmpl", struct{ Settings string }{settings})
}

// lines is the include of /etc/ws/gitconfig in ~/.gitconfig.
func (g Config) lines() file.HomeLines {
	return g.A.Lines(g.M, ".gitconfig", file.UserLines{
		Lines: gitcfg.Include, Top: true, Mode: 0o644,
		Shown: "[include] path = " + gitcfg.Path,
		Name:  "the include of " + gitcfg.Path,
		Has:   g.included,
	})
}

// Plan plans /etc/ws/gitconfig, then the include in ~/.gitconfig. Where it plans the include and the
// file has settings of identity or signing of its own, or git's XDG file has, it warns that they win
// or lose. Each run, with the include or without, it warns of a credential section of the file that
// resets the helper of a token.
func (g Config) Plan(ctx context.Context) ([]resource.Change, error) {
	content, err := g.content()
	if err != nil {
		return nil, err
	}
	changes, err := plan.All(ctx, etcws.Files(g.M, gitcfg.Path, content)...)
	if err != nil {
		return nil, err
	}
	include, err := g.lines().Plan(ctx)
	if err != nil {
		return nil, err
	}
	if err := g.note(ctx, len(include) > 0); err != nil {
		return nil, err
	}
	return append(changes, include...), nil
}

// note warns of what in the user's files shadows ws's (shadow.notes). A missing ~/.gitconfig warns
// of nothing.
func (g Config) note(ctx context.Context, adding bool) error {
	text, err := g.A.ReadHome(g.M, ".gitconfig")
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	notes, err := shadow{M: g.M, A: g.A, Creds: g.Creds, Whole: g.Whole}.notes(ctx, text, adding)
	for _, n := range notes {
		g.Out.Warn("%s", n)
	}
	return err
}

// included says whether the text of ~/.gitconfig includes /etc/ws/gitconfig (Included). An
// include out of place is file.Misplaced's error, which the user's lines refuse.
func (g Config) included(ctx context.Context, m effect.Machine, path string, text []byte) (bool, error) {
	holds, err := Included(ctx, m, path, text)
	var out misplaced
	if errors.As(err, &out) {
		return false, file.Misplaced("%v", err)
	}
	return holds, err
}

// configCheck is the command that has git read a config file before it replaces the old one.
func configCheck(path string) (string, []string) {
	return "git", []string{"config", "--file", path, "--name-only", "--list"}
}
