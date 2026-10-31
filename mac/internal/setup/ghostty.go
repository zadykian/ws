package setup

import (
	"context"
	"errors"
	"os/exec"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

const (
	// ghosttyFont is the cask of the font that ws.ghostty names, and ghosttyFontFile the file of its
	// regular style.
	ghosttyFont     = "font-jetbrains-mono-nerd-font"
	ghosttyFontFile = "JetBrainsMonoNerdFontMono-Regular.ttf"
)

// Ghostty sets up Ghostty, the Mac's terminal: Ghostty and the font its settings name, from
// Homebrew's casks where missing, and ws.ghostty, those settings, which Ghostty's config reads.
// It warns where Ghostty can't be installed, or Ghostty's config is not the step's to change, so
// that the run goes on to the tunnel. It stops where it can't write a file. Ghostty then updates
// itself, as ws.ghostty says.
type Ghostty struct {
	Home string
	// Settings is the path of mac/ws.ghostty.
	Settings string
	M        effect.Machine
	Out      effect.Out
	// LookPath finds a program on the user's PATH. Nil means exec.LookPath.
	LookPath func(string) (string, error)
	// Real resolves the links of a directory. Nil means filepath.EvalSymlinks.
	Real   func(string) (string, error)
	failed map[string]bool // the casks whose install failed, so the plan skips them for the run
}

// Name is ghostty.
func (*Ghostty) Name() string { return "ghostty" }

// WaitsOn is nothing.
func (*Ghostty) WaitsOn() string { return "" }

// has reports whether the first of the files is there, in /Applications or ~/Applications for the
// app, where its cask puts it, or in ~/Library/Fonts or /Library/Fonts for the font.
func (g *Ghostty) has(name string, dirs ...string) bool {
	for _, d := range dirs {
		if _, err := g.M.Stat(d + "/" + name); err == nil {
			return true
		}
	}
	return false
}

// cask plans Homebrew's cask where there is none of its files, and says whether they are there,
// or would be after the plan. Where Homebrew has the cask but its files are gone, it warns instead,
// as gateway does: brew install would upgrade the cask. Without Homebrew it stops.
func (g *Ghostty) cask(name string, there bool, missing string) ([]step.Change, bool, error) {
	if there {
		return nil, true, nil
	}
	if g.failed[name] {
		return nil, false, nil
	}
	look := g.LookPath
	if look == nil {
		look = exec.LookPath
	}
	if _, err := look("brew"); err != nil {
		return nil, false, errors.New("install Homebrew first: https://brew.sh")
	}
	if _, err := g.M.Run(context.Background(), "brew", "list", "--cask", name); err != nil {
		return []step.Change{Run{Name: "brew", Args: []string{"install", "--cask", name}}}, true, nil
	}
	g.Out.Warn("Homebrew has the cask %s, but %s: brew reinstall --cask %s puts it back",
		name, missing, name)
	return nil, false, nil
}

// Plan plans the app, then the font where the app is there or comes.
func (g *Ghostty) Plan(_ context.Context) (step.Plan, error) {
	app, ok, err := g.cask("ghostty", g.has("Ghostty.app", "/Applications", g.Home+"/Applications"),
		"no Ghostty.app is in /Applications or ~/Applications")
	if err != nil || !ok {
		return step.Plan{}, err
	}
	font, _, err := g.cask(ghosttyFont, g.has(ghosttyFontFile, g.Home+"/Library/Fonts", "/Library/Fonts"),
		"no "+ghosttyFontFile+" is in ~/Library/Fonts or /Library/Fonts")
	if err != nil {
		return step.Plan{}, err
	}
	return step.Plan{Changes: append(app, font...)}, nil
}
