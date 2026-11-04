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

// config is config.json. The false fields keep no omitempty: a missing key brings Fresh's default
// back, which checks for updates, reports telemetry and execs a daemon.
type config struct {
	CheckForUpdates bool           `json:"check_for_updates"`
	SelfUpdate      bool           `json:"self_update"`
	AutoAttach      bool           `json:"auto_attach"`
	Editor          editorSettings `json:"editor"`
}

// Config renders config.json.
func Config() ([]byte, error) {
	settings, err := editor.LoadSettings()
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString(header)
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(config{
		Editor: editorSettings{
			Rulers:         []int{settings.Ruler},
			WhitespaceShow: settings.VisibleWhitespace, SpacesLeading: settings.VisibleWhitespace,
			SpacesInner: settings.VisibleWhitespace, SpacesTrailing: settings.VisibleWhitespace,
			TabsInner: settings.VisibleWhitespace, TabsTrailing: settings.VisibleWhitespace,
			AutoSave: settings.AutoSaveSeconds > 0, AutoSaveInterval: settings.AutoSaveSeconds,
		},
	})
	return out.Bytes(), err
}
