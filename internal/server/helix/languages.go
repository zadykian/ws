package helix

import (
	"embed"
	"encoding/json"

	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/devtools"
	"github.com/zadykian/ws/internal/server/values"
)

//go:embed files
var files embed.FS

// quote returns the string as a JSON string, which is a TOML basic string too.
func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// languages renders languages.toml: the language servers by path.
func languages(v values.Values) ([]byte, error) {
	return host.Render(files, "files/languages.toml.tmpl", struct{ Gopls, Roslyn string }{
		Gopls:  quote(devtools.Gopls(v.Home)),
		Roslyn: quote(devtools.Roslyn(v.Home)),
	})
}
