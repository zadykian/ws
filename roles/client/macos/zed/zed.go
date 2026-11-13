// Package zed installs Zed and its font on the Mac, and writes the settings and keymap that Zed
// reads.
package zed

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/internal/brew"
)

// Zed sets up Zed, the Mac's editor for ws. It installs Zed and the font its settings name from
// Homebrew's casks where missing, and writes ws's settings and keymap in ~/.config/zed. It warns
// where Zed can't be installed, so that the run goes on to the tunnel. Zed then updates itself.
type Zed struct {
	Home string
	// Settings is global_settings.json whole, its first line zedHeader.
	Settings []byte
	// Keymap is keymap.json whole, its first line starting with zedKeymapMark.
	Keymap []byte
	M      effect.Machine
	Out    effect.Out
	// LookPath finds a program on the user's PATH. Nil means exec.LookPath.
	LookPath func(string) (string, error)
	failed   map[string]bool // the casks whose install failed, so the plan skips them for the run
}

// casks is the cask code that Zed shares with Ghostty, built from the step's fields.
func (z *Zed) casks() brew.Casks {
	if z.failed == nil {
		z.failed = map[string]bool{}
	}
	return brew.Casks{M: z.M, Out: z.Out, LookPath: z.LookPath, Failed: z.failed, App: "zed"}
}
