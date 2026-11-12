package fresh

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/internal/server/values"
)

// configFiles returns ~/.config/fresh with config.json and the theme, which ws replaces whole. The
// directories stay the user's, as Fresh writes its own files there. The old file stays beside the
// new one as a backup: Fresh's scan of themes/ takes only .json, so a backup isn't a theme.
func configFiles(m effect.Machine, v values.Values, a host.Account) ([]host.Resource, error) {
	config, err := host.Render(templates, "files/config.json.tmpl", struct{ Home string }{v.Home})
	if err != nil {
		return nil, err
	}
	settings := a.File(m, ".config/fresh/config.json", config, 0o644)
	theme := a.File(m, ".config/fresh/themes/jetbrains-dark.json", themeFile, 0o644)
	settings.Backup, theme.Backup = true, true
	return []host.Resource{settings, theme}, nil
}
