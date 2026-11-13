package fresh

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// configFiles returns ~/.config/fresh with config.json and the theme, which ws replaces whole. The
// directories stay the user's, as Fresh writes its own files there. The old file stays beside the
// new one as a backup: Fresh's scan of themes/ takes only .json, so a backup isn't a theme.
func configFiles(m effect.Machine, v values.Values, a account.Account) ([]resource.Resource, error) {
	config, err := resource.Render(templates, "files/config.json.tmpl", struct{ Home string }{v.Home})
	if err != nil {
		return nil, err
	}
	settings := a.File(m, ".config/fresh/config.json", config, 0o644)
	theme := a.File(m, ".config/fresh/themes/jetbrains-dark.json", themeFile, 0o644)
	settings.Backup, theme.Backup = true, true
	return []resource.Resource{settings, theme}, nil
}
