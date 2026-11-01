package shell

import (
	"context"

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

// Plan plans the file. Git must accept the new file
// before it replaces the old one.
func (g gitconfig) Plan(ctx context.Context) ([]host.Change, error) {
	content, err := g.content()
	if err != nil {
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
