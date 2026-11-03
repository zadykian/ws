// Package zed is the zed role as Go steps: the settings of Zed's server for Go and C#, and the
// open-file limit of ws_user's logins. See docs/zed.md.
package zed

import (
	"embed"
	"encoding/json"

	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/devtools"
	"github.com/zadykian/ws/internal/server/values"
)

//go:embed settings.json.tmpl
var files embed.FS

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
