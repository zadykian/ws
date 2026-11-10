package shell

import (
	"context"
	"embed"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

//go:embed files
var files embed.FS

// dotnetArch is the architecture in .NET's portable runtime ID: arm64, or x64 for the rest.
func dotnetArch(goarch string) string {
	if goarch == "arm64" {
		return "arm64"
	}
	return "x64"
}

// dotfiles returns ~/.bashrc and ~/.profile, which ws replaces whole: settings of the user's own
// go in ~/.bash_aliases, which ~/.bashrc sources. goarch is the machine's architecture: ws is built
// for the machine it runs on.
func dotfiles(m effect.Machine, a account, goarch string) ([]host.Resource, error) {
	rc, err := host.Render(files, "files/bashrc.tmpl", struct{ Arch string }{dotnetArch(goarch)})
	if err != nil {
		return nil, err
	}
	profile, err := files.ReadFile("files/profile")
	if err != nil {
		return nil, err
	}
	return []host.Resource{
		host.File{M: m, Path: a.Home + "/.bashrc", Content: rc, Mode: 0o644, UID: a.UID, GID: a.GID},
		host.File{M: m, Path: a.Home + "/.profile", Content: profile, Mode: 0o644, UID: a.UID, GID: a.GID},
	}, nil
}

// owned makes the step's parts once the account is found. The user's uid and gid come from the
// machine, and a dry run on a new one must still show them.
func owned(user host.User, m effect.Machine, title, after string,
	parts func(a account) ([]host.Resource, error)) host.Lazy {
	return host.Lazy{Title: title, After: after, Parts: func(ctx context.Context) ([]host.Resource, error) {
		a, err := lookup(ctx, m, user)
		if err != nil {
			return nil, err
		}
		return parts(a)
	}}
}
