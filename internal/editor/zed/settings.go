package zed

import (
	"embed"

	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/editor"
)

//go:embed global_settings.json.tmpl
var templates embed.FS

// Settings renders the Mac's global_settings.json, under the header that ws mac apply adds.
func Settings() ([]byte, error) {
	settings, err := editor.LoadSettings()
	if err != nil {
		return nil, err
	}
	return host.Render(templates, "global_settings.json.tmpl", struct {
		Mode, Light, Dark, Font string
	}{
		Mode:  quote(settings.Theme.Mode),
		Light: quote(settings.Theme.Light),
		Dark:  quote(settings.Theme.Dark),
		Font:  quote(settings.Font),
	})
}
