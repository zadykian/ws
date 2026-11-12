package shell

import (
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
func dotfiles(m effect.Machine, a host.Account, goarch string) ([]host.Resource, error) {
	rc, err := host.Render(files, "files/bashrc.tmpl", struct{ Arch string }{dotnetArch(goarch)})
	if err != nil {
		return nil, err
	}
	profile, err := files.ReadFile("files/profile")
	if err != nil {
		return nil, err
	}
	return []host.Resource{
		a.File(m, ".bashrc", rc, 0o644),
		a.File(m, ".profile", profile, 0o644),
	}, nil
}
