// Package ghostty picks the config of Ghostty's that the line reading /etc/ws/ghostty goes in, a
// fact of the Mac's units. Its units.toml.tmpl sets Ghostty up.
package ghostty

import "github.com/zadykian/ws/engine/core/effect"

// ghosttyConfigs lists the config files that Ghostty reads under home, in the order the menu's Open
// Config tries them (src/config/edit.zig, 1.3.1 and main). They are config.ghostty and config in
// Application Support's com.mitchellh.ghostty, then in ~/.config/ghostty. Ghostty's app, which
// launchd starts, sees no XDG_CONFIG_HOME of a shell's.
func ghosttyConfigs(home string) []string {
	support := home + "/Library/Application Support/com.mitchellh.ghostty"
	return []string{
		support + "/config.ghostty", support + "/config",
		home + "/.config/ghostty/config.ghostty", home + "/.config/ghostty/config",
	}
}

// ConfigPath returns the config file that Ghostty's menu opens, where the line that reads
// /etc/ws/ghostty goes. That is the first of ghosttyConfigs that holds anything, else the first
// that exists, else the first. The pick stays once the line is in, as that file then holds it.
func ConfigPath(m effect.Machine, home string) string {
	configs := ghosttyConfigs(home)
	found := ""
	for _, c := range configs {
		info, err := m.Stat(c)
		if err != nil || !info.Mode.IsRegular() {
			continue
		}
		if text, err := m.ReadFile(c); err != nil || len(text) > 0 {
			return c
		}
		if found == "" {
			found = c
		}
	}
	if found == "" {
		return configs[0]
	}
	return found
}
