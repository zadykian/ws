package shell

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// gitconfig is ~/.gitconfig, which ws replaces whole. A real run keeps the old file beside it, as
// ~/.gitconfig.*~. A diff would print a credential that a file set up by hand holds, such as an
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
		Check: configCheck,
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
	if g.Signing {
		if err := g.signingNotes(ctx, old); err != nil {
			return err
		}
	}
	oldKeys, err := gitNames(ctx, g.M, old)
	if err != nil {
		return err
	}
	wantKeys, err := gitNames(ctx, g.M, content)
	if err != nil {
		return err
	}
	if lost := dropped(oldKeys, wantKeys); len(lost) > 0 {
		g.Out.Warn("writing ~/.gitconfig drops %s. Move them to ~/.config/git/local first; a real run keeps the old file beside it, as ~/.gitconfig.*~.", join(lost))
	}
	return nil
}

// signingNotes names the signing key that the rewrite replaces, which the note on dropped keys
// misses as the key's name stays. It also names the signing settings of ~/.config/git/local, which
// win over the role's.
func (g gitconfig) signingNotes(ctx context.Context, old []byte) error {
	list, err := g.M.RunInput(ctx, old, "git", "config", "--file", "-", "--list")
	if err != nil {
		return err
	}
	if key, ok := signingKeyOf(lines(list), g.A.Home); ok && key != g.Key+".pub" {
		g.Out.Warn("writing ~/.gitconfig replaces its signing key, %s, with %s.pub. To sign with the old key, set user.signingkey in ~/.config/git/local first.", key, g.Key)
	}
	local, err := g.M.ReadRootFile(g.A.Home + "/.config/git/local")
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	keys, err := gitNames(ctx, g.M, local)
	if err != nil {
		return err
	}
	if found := localSigning(keys); len(found) > 0 {
		g.Out.Warn("~/.config/git/local sets %s, which win over the role's signing in ~/.gitconfig. Remove them to sign with %s.", join(found), g.Key)
	}
	return nil
}

// configCheck is the command that has git read a config file before it replaces the old one.
func configCheck(path string) (string, []string) {
	return "git", []string{"config", "--file", path, "--name-only", "--list"}
}
