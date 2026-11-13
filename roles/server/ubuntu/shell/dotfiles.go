package shell

import (
	"embed"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
)

//go:embed files/bashrc.tmpl files/profile
var files embed.FS

// dotnetArch is the architecture in .NET's portable runtime ID: arm64, or x64 for the rest. It
// takes dpkg's name.
func dotnetArch(arch string) string {
	if arch == "arm64" {
		return "arm64"
	}
	return "x64"
}

// dotfiles returns ~/.bashrc and ~/.profile, which ws replaces whole: settings of the user's own
// go in ~/.bash_aliases, which ~/.bashrc sources. arch is the machine's architecture as dpkg names it.
func dotfiles(m effect.Machine, a account.Account, arch string) ([]resource.Resource, error) {
	rc, err := resource.Render(files, "files/bashrc.tmpl", struct{ Arch string }{dotnetArch(arch)})
	if err != nil {
		return nil, err
	}
	profile, err := files.ReadFile("files/profile")
	if err != nil {
		return nil, err
	}
	return []resource.Resource{
		a.File(m, ".bashrc", rc, 0o644),
		a.File(m, ".profile", profile, 0o644),
	}, nil
}
