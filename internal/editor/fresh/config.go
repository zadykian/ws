// Package fresh turns the neutral data of internal/editor into the files that the Fresh editor
// reads. See docs/fresh.md.
package fresh

import (
	"bytes"
	"encoding/json"

	"github.com/zadykian/ws/internal/editor"
)

// header is allowed, as Fresh reads config.json as JSONC.
const header = "// Written by github.com/zadykian/ws, whose next run replaces any change made here.\n"

// editorSettings is the editor table. Rainbow brackets stay off: the theme has no place for their
// colours.
type editorSettings struct {
	Rulers           []int `json:"rulers"`
	WhitespaceShow   bool  `json:"whitespace_show"`
	SpacesLeading    bool  `json:"whitespace_spaces_leading"`
	SpacesInner      bool  `json:"whitespace_spaces_inner"`
	SpacesTrailing   bool  `json:"whitespace_spaces_trailing"`
	TabsInner        bool  `json:"whitespace_tabs_inner"`
	TabsTrailing     bool  `json:"whitespace_tabs_trailing"`
	AutoSave         bool  `json:"auto_save_enabled"`
	AutoSaveInterval int   `json:"auto_save_interval_secs"`
	RainbowBrackets  bool  `json:"rainbow_brackets"`
}

// keybindingMap is a named map of bindings. It inherits Fresh's default for every key it leaves out.
type keybindingMap struct {
	Inherits string    `json:"inherits"`
	Bindings []binding `json:"bindings"`
}

// config is config.json. The false fields keep no omitempty: a missing key brings Fresh's default
// back, which checks for updates, reports telemetry and execs a daemon.
type config struct {
	Theme           string                   `json:"theme"`
	CheckForUpdates bool                     `json:"check_for_updates"`
	SelfUpdate      bool                     `json:"self_update"`
	AutoAttach      bool                     `json:"auto_attach"`
	ActiveMap       string                   `json:"active_keybinding_map"`
	Maps            map[string]keybindingMap `json:"keybinding_maps"`
	Keybindings     []binding                `json:"keybindings"`
	Editor          editorSettings           `json:"editor"`
	Languages       map[string]language      `json:"languages"`
	LSP             map[string][]server      `json:"lsp"`
}

// Config renders config.json. paths gives each server of the catalogue its binary.
func Config(paths map[string]string) ([]byte, error) {
	keymap, err := editor.LoadKeymap()
	if err != nil {
		return nil, err
	}
	targets, err := editor.LoadTargets(actionFiles, "actions/*.yaml")
	if err != nil {
		return nil, err
	}
	if err := keymap.Check(targets); err != nil {
		return nil, err
	}
	rendered, err := bindings(keymap, targets)
	if err != nil {
		return nil, err
	}
	mapped, outside := split(rendered)
	settings, err := editor.LoadSettings()
	if err != nil {
		return nil, err
	}
	catalogue, err := editor.LoadServers()
	if err != nil {
		return nil, err
	}
	servers, err := serverEntries(catalogue, paths)
	if err != nil {
		return nil, err
	}
	types, err := editor.LoadFileTypes()
	if err != nil {
		return nil, err
	}
	langs, err := languageEntries(types)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString(header)
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(config{
		Theme:       settings.Theme.Dark,
		ActiveMap:   "f12",
		Maps:        map[string]keybindingMap{"f12": {Inherits: "default", Bindings: mapped}},
		Keybindings: outside,
		Editor: editorSettings{
			Rulers:         []int{settings.Ruler},
			WhitespaceShow: settings.VisibleWhitespace, SpacesLeading: settings.VisibleWhitespace,
			SpacesInner: settings.VisibleWhitespace, SpacesTrailing: settings.VisibleWhitespace,
			TabsInner: settings.VisibleWhitespace, TabsTrailing: settings.VisibleWhitespace,
			AutoSave: settings.AutoSaveSeconds > 0, AutoSaveInterval: settings.AutoSaveSeconds,
		},
		Languages: langs,
		LSP:       servers,
	})
	return out.Bytes(), err
}
