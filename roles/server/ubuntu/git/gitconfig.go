// Package git is the git role as Go steps: ~/.gitconfig, the key that signs commits, the allowed
// signers, the tokens for servers besides GitHub and the clones. See roles/server/ubuntu/git/README.md.
package git

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"

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

// note warns of what in the user's files shadows ws's. Where the run adds the include, it names
// the settings of identity and signing in ~/.gitconfig, which win over /etc/ws/gitconfig's below
// the include. It names those in git's XDG file too, which lose to them. On each run, it names
// the credential sections of ~/.gitconfig that reset a token's helper. A missing ~/.gitconfig
// warns of nothing. The notes name settings and sections, never a value.
func (g Config) note(ctx context.Context, adding bool) error {
	text, err := g.A.ReadHome(g.M, ".gitconfig")
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if adding {
		names, err := gitNames(ctx, g.M, text)
		if err != nil {
			return err
		}
		if own := gitcfg.Own(names); own != "" {
			g.Out.Warn("~/.gitconfig's own settings of %s, below the include, win over %s's", own, gitcfg.Path)
		}
		if err := g.noteXDG(ctx); err != nil {
			return err
		}
	}
	return g.noteCredentials(ctx, text)
}

// noteXDG warns of the settings of identity and signing in git's XDG file, which git reads before
// ~/.gitconfig, so that /etc/ws/gitconfig's win over them. It reads below the home alone, as the
// run has no XDG_CONFIG_HOME of the account's.
func (g Config) noteXDG(ctx context.Context) error {
	rel, err := filepath.Rel(g.A.Home, gitcfg.XDGFile("", g.A.Home))
	if err != nil {
		return err
	}
	text, err := g.A.ReadHome(g.M, rel)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	names, err := gitNames(ctx, g.M, text)
	if err != nil {
		return err
	}
	if own := gitcfg.Own(names); own != "" {
		g.Out.Warn("%s", gitcfg.XDGNote(g.A.Home+"/"+rel, own, gitcfg.Path))
	}
	return nil
}

// included says whether the text of ~/.gitconfig includes /etc/ws/gitconfig, as git reads it. Git
// reads the text on its standard input, so that the plan holds the file's bytes alone.
func (g Config) included(ctx context.Context, m effect.Machine, path string, text []byte) (bool, error) {
	out, err := m.RunInput(ctx, text, "git", "config", "--file", "-", "--type", "path", "--get-all", "include.path")
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return false, fmt.Errorf("git config cannot read %s: %w", path, err)
	}
	names, err := gitNames(ctx, m, text)
	if err != nil {
		return false, err
	}
	first := ""
	if len(names) > 0 {
		first = names[0]
	}
	holds, err := gitcfg.Included(lines(out), first)
	if err != nil {
		return false, file.Misplaced("%s %v", path, err)
	}
	return holds, nil
}

// configCheck is the command that has git read a config file before it replaces the old one.
func configCheck(path string) (string, []string) {
	return "git", []string{"config", "--file", path, "--name-only", "--list"}
}
