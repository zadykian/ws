package shell

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// gitconfig is ~/.gitconfig, which ws replaces whole. A real run keeps the old file beside it, as
// ~/.gitconfig.*~, and a diff would print a credential that a file set up by hand holds, such as an
// http.extraHeader, so it shows none. The settings of the machine's own go in ~/.config/git/local,
// which the file includes last.
type gitconfig struct {
	M       effect.Machine
	Out     effect.Out
	A       account
	Name    string
	Email   string
	Signing bool
	Key     string
	Signers string
}

func (g gitconfig) path() string { return g.A.Home + "/.gitconfig" }

func (g gitconfig) content() ([]byte, error) {
	return host.Render(files, "files/gitconfig.tmpl", struct {
		Name, Email, Key, Signers string
		Signing                   bool
	}{g.Name, g.Email, g.Key, g.Signers, g.Signing})
}

// names lists the keys of a config's content, by git, without their values.
func (g gitconfig) names(ctx context.Context, content []byte) ([]string, error) {
	out, err := g.M.RunInput(ctx, content, "git", "config", "--file", "-", "--name-only", "--list")
	return lines(out), err
}

// Plan says what the rewrite loses or replaces, then plans the file. Git must accept the new file
// before it replaces the old one.
func (g gitconfig) Plan(ctx context.Context) ([]host.Change, error) {
	content, err := g.content()
	if err != nil {
		return nil, err
	}
	if err := g.notes(ctx, content); err != nil {
		return nil, err
	}
	return host.Validated{
		File: host.File{
			M: g.M, Path: g.path(), Content: content, Mode: 0o600, UID: g.A.UID, GID: g.A.GID,
			Secret: true, Backup: true,
		},
		Check: func(path string) (string, []string) {
			return "git", []string{"config", "--file", path, "--name-only", "--list"}
		},
	}.Plan(ctx)
}

func (g gitconfig) notes(ctx context.Context, content []byte) error {
	old, err := g.M.ReadRootFile(g.path())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	oldKeys, err := g.names(ctx, old)
	if err != nil {
		return err
	}
	wantKeys, err := g.names(ctx, content)
	if err != nil {
		return err
	}
	if lost := dropped(oldKeys, wantKeys); len(lost) > 0 {
		g.Out.Warn("writing ~/.gitconfig drops %s. Move them to ~/.config/git/local first; a real run keeps the old file beside it, as ~/.gitconfig.*~.", join(lost))
	}
	return nil
}
