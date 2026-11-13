// Package zed installs Zed and its font on the Mac, and writes the settings and keymap that Zed
// reads.
package zed

import (
	"context"
	"github.com/zadykian/ws/engine/core/resource/file"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
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

// plan plans the app, then the font where the app is there or comes, and the files.
func (z *Zed) plan(ctx context.Context) (step.Plan, error) {
	b := z.casks()
	app, ok, err := b.Cask(ctx, "zed", b.Has("Zed.app", "/Applications", z.Home+"/Applications"),
		"no Zed.app is in /Applications or ~/Applications")
	if err != nil || !ok {
		return step.Plan{}, err
	}
	font, err := b.Font(ctx, z.Home)
	if err != nil {
		return step.Plan{}, err
	}
	files, err := z.files(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	return step.Plan{Changes: append(append(app, font...), files...)}, nil
}

// report says how the casks and the files stand. settings.json is Zed's own, which ws writes once
// and never reads, so a report only finds it.
func (z *Zed) report(ctx context.Context) ([]step.Item, error) {
	dir := z.Home + "/.config/zed"
	files := z.whole()
	return file.Gather(ctx, z.casks().Report(ctx, "zed", "Zed.app", z.Home),
		z.dirs(), files[0], files[1],
		file.UserPath{M: z.M, Kind: step.FileItem, Path: dir + "/settings.json"})
}
