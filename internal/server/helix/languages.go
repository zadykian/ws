package helix

import (
	"embed"
	"encoding/json"
	"strings"

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

// languages renders languages.toml: the grammars to build, and the language servers by path.
func languages(v values.Values) ([]byte, error) {
	names := make([]string, len(v.Helix.Grammars))
	for i, g := range v.Helix.Grammars {
		names[i] = quote(g)
	}
	return host.Render(files, "files/languages.toml.tmpl", struct{ Grammars, Gopls, Roslyn string }{
		Grammars: "[" + strings.Join(names, ", ") + "]",
		Gopls:    quote(devtools.Gopls(v.Home)),
		Roslyn:   quote(devtools.Roslyn(v.Home)),
	})
}
