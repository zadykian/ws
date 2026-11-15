package shell

import (
	"context"
	"embed"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/roles/internal/etcws"
)

//go:embed files/bashrc.tmpl files/profile
var files embed.FS

// Ws's bash files, which the user's ~/.bashrc and ~/.profile source.
const (
	bashrcPath  = etcws.Dir + "/bashrc"
	profilePath = etcws.Dir + "/profile"
)

// dotnetArch is the architecture in .NET's portable runtime ID: arm64, or x64 for the rest. It
// takes dpkg's name.
func dotnetArch(arch string) string {
	if arch == "arm64" {
		return "arm64"
	}
	return "x64"
}

// sources says whether a line of text is line, spaces aside.
func sources(line string) func(context.Context, effect.Machine, string, []byte) (bool, error) {
	return func(_ context.Context, _ effect.Machine, _ string, text []byte) (bool, error) {
		for _, l := range strings.Split(string(text), "\n") {
			if strings.TrimSpace(l) == line {
				return true, nil
			}
		}
		return false, nil
	}
}

// userLine is the line that sources path, at the top of the user's file. Stock's guard of a shell
// that is not interactive returns first, and ~/.profile runs ~/.bashrc before its end. The line
// uses `.`, not `source`, as sh reads ~/.profile too.
func userLine(path string) file.UserLines {
	line := ". " + path
	return file.UserLines{Lines: line + "\n", Shown: line, Top: true, Mode: 0o644, Has: sources(line)}
}

// dotfiles returns ws's bash lines in /etc/ws/bashrc and /etc/ws/profile, root's. Then it returns
// the line that sources each at the top of the user's ~/.bashrc and ~/.profile, which stay the
// user's. The files come first, so that a line never names a missing file. arch is the machine's
// architecture as dpkg names it.
func dotfiles(m effect.Machine, a account.Account, arch string) ([]resource.Resource, error) {
	rc, err := resource.Render(files, "files/bashrc.tmpl", struct{ Arch string }{dotnetArch(arch)})
	if err != nil {
		return nil, err
	}
	profile, err := files.ReadFile("files/profile")
	if err != nil {
		return nil, err
	}
	parts := append(etcws.Files(m, bashrcPath, rc), etcws.Inside(m, profilePath, profile)...)
	return append(parts, a.Lines(m, ".bashrc", userLine(bashrcPath)), a.Lines(m, ".profile", userLine(profilePath))), nil
}
