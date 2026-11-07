package setup

import (
	"context"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// Zed sets up Zed, the Mac's editor for ws: Zed and the font its settings name, from Homebrew's
// casks where missing, and ws's settings and keymap in ~/.config/zed. It warns where Zed can't be installed,
// so that the run goes on to the tunnel. Zed then updates itself.
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

// Name is zed.
func (*Zed) Name() string { return "zed" }

// WaitsOn is nothing.
func (*Zed) WaitsOn() string { return "" }

// casks is the cask code that Zed shares with Ghostty, built from the step's fields.
func (z *Zed) casks() brewCasks {
	return brewCasks{m: z.M, out: z.Out, lookPath: z.LookPath, failed: z.failed}
}

// Plan plans the app, then the font where the app is there or comes, and the files.
func (z *Zed) Plan(ctx context.Context) (step.Plan, error) {
	b := z.casks()
	app, ok, err := b.cask(ctx, "zed", b.has("Zed.app", "/Applications", z.Home+"/Applications"),
		"no Zed.app is in /Applications or ~/Applications")
	if err != nil || !ok {
		return step.Plan{}, err
	}
	font, err := b.font(ctx, z.Home)
	if err != nil {
		return step.Plan{}, err
	}
	return step.Plan{Changes: append(append(app, font...), z.files()...)}, nil
}

// Report says how the casks and the files stand. settings.json is Zed's own, which ws writes once
// and never reads, so a report only finds it.
func (z *Zed) Report(ctx context.Context) ([]step.Item, error) {
	dir := z.Home + "/.config/zed"
	return gather(ctx, z.casks().report(ctx, "zed", "Zed.app", z.Home),
		userPath{M: z.M, Kind: step.DirItem, Path: dir},
		userFile{M: z.M, Path: dir + "/global_settings.json", Text: z.Settings, Mode: 0o644, Own: zedOwn(zedHeader, true)},
		userFile{M: z.M, Path: dir + "/keymap.json", Text: z.Keymap, Mode: 0o644, Own: zedOwn(zedKeymapMark, false)},
		userPath{M: z.M, Kind: step.FileItem, Path: dir + "/settings.json"})
}

// Apply installs the casks. A failed install is a warning, and the next run tries again; where
// the app's fails, the files wait for it.
func (z *Zed) Apply(ctx context.Context, p step.Plan) error {
	if z.failed == nil {
		z.failed = map[string]bool{}
	}
	return z.casks().install(ctx, p, "zed")
}
