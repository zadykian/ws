// Package zed is the zed role as Go steps: the settings of Zed's server for Go and C#, and the
// open-file limit of ws_user's logins. See docs/zed.md.
package zed

import (
	"embed"
	"encoding/json"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/devtools"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

//go:embed settings.json.tmpl
var files embed.FS

// Steps returns the role's steps in order.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, _ effect.Out) []step.Step {
	user := host.User{Name: v.User, Home: v.Home}
	return []step.Step{
		user.Owned(m, "Zed settings", "", func(uid, gid int) ([]host.Resource, error) {
			settings, err := settingsFile(v)
			if err != nil {
				return nil, err
			}
			dir := v.Home + "/.config/zed"
			return []host.Resource{
				host.Dir{M: m, Path: v.Home + "/.config", Mode: 0o755, UID: uid, GID: gid, CreateOnly: true},
				host.Dir{M: m, Path: dir, Mode: 0o755, UID: uid, GID: gid},
				host.File{M: m, Path: dir + "/settings.json", Content: settings, Mode: 0o644, UID: uid, GID: gid,
					Backup: true},
			}, nil
		}),
	}
}

// quote returns the string as a JSON string.
func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// settingsFile renders ~/.config/zed/settings.json: the servers by path, as devtools installs
// them. The server's file wins over the Mac's, which Zed forwards.
func settingsFile(v values.Values) ([]byte, error) {
	return host.Render(files, "settings.json.tmpl", struct{ Gopls, Roslyn, Dlv string }{
		Gopls:  quote(devtools.Gopls(v.Home)),
		Roslyn: quote(devtools.Roslyn(v.Home)),
		Dlv:    quote(devtools.Dlv(v.Home)),
	})
}
