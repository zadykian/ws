// Package fresh turns the neutral data of internal/editor into the files that the Fresh editor
// reads. See docs/fresh.md.
package fresh

import (
	"bytes"
	"encoding/json"
)

// header is allowed, as Fresh reads config.json as JSONC.
const header = "// Written by github.com/zadykian/ws, whose next run replaces any change made here.\n"

// config is config.json. The false fields keep no omitempty: a missing key brings Fresh's default
// back, which checks for updates, reports telemetry and execs a daemon.
type config struct {
	CheckForUpdates bool `json:"check_for_updates"`
	SelfUpdate      bool `json:"self_update"`
	AutoAttach      bool `json:"auto_attach"`
}

// Config renders config.json.
func Config() ([]byte, error) {
	var out bytes.Buffer
	out.WriteString(header)
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(config{})
	return out.Bytes(), err
}
