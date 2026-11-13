// Package brew plans and installs the Homebrew casks of an app and its font, for the roles that
// set up an app.
package brew

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
	"github.com/zadykian/ws/roles/client/macos/internal/userfile"
)

const (
	// Font is the cask of the font that Ghostty's and Zed's settings name.
	Font = "font-jetbrains-mono-nerd-font"
	// FontFile is the file of the font's regular style.
	FontFile = "JetBrainsMonoNerdFontMono-Regular.ttf"
	// FontGone is why the font counts as gone.
	FontGone = "no " + FontFile + " is in ~/Library/Fonts or /Library/Fonts"
)

// Casks plans and installs an app's Homebrew cask and the font's, for the steps that set up an
// app. A step builds it per call from its own fields, and the Failed map is shared with the step.
type Casks struct {
	M        effect.Machine
	Out      effect.Out
	LookPath func(string) (string, error)
	Failed   map[string]bool // the casks whose install failed, so the plan skips them for the run
}

// Has reports whether the first of the files is there. An app's cask puts it in /Applications or
// ~/Applications, and a font's in ~/Library/Fonts or /Library/Fonts.
func (b Casks) Has(name string, dirs ...string) bool {
	for _, d := range dirs {
		if _, err := b.M.Stat(d + "/" + name); err == nil {
			return true
		}
	}
	return false
}

// Cask plans Homebrew's cask where there is none of its files, and says whether they are there,
// or would be after the plan. Where Homebrew has the cask but its files are gone, it warns instead:
// brew install would upgrade the cask. Without Homebrew it stops.
func (b Casks) Cask(ctx context.Context, name string, there bool, missing string) ([]step.Change, bool, error) {
	if there {
		return nil, true, nil
	}
	if b.Failed[name] {
		return nil, false, nil
	}
	look := b.LookPath
	if look == nil {
		look = exec.LookPath
	}
	if _, err := look("brew"); err != nil {
		return nil, false, errors.New("install Homebrew first: https://brew.sh")
	}
	if _, err := b.M.Run(ctx, "brew", "list", "--cask", name); err != nil {
		return []step.Change{change.Run{Name: "brew", Args: []string{"install", "--cask", name}}}, true, nil
	}
	b.Out.Warn("%s", CaskGone(name, missing))
	return nil, false, nil
}

// CaskGone says why a cask that Homebrew has is missing from the Mac.
func CaskGone(name, missing string) string {
	return fmt.Sprintf("Homebrew has the cask %s, but %s: brew reinstall --cask %s puts it back",
		name, missing, name)
}

// Item says how the cask stands, as cask reads it. brew is asked of one cask, and only where its
// files are gone: a report covers what ws manages, never all that Homebrew holds.
func (b Casks) Item(ctx context.Context, name string, there bool, missing string) step.Item {
	item := step.Item{Kind: step.PackageItem, Name: name, Want: userfile.StateInstalled, Found: userfile.StateInstalled}
	if there {
		return item
	}
	item.Found = userfile.StateMissing
	if _, err := b.M.Run(ctx, "brew", "list", "--cask", name); err == nil {
		item.Conflict = CaskGone(name, missing)
	}
	return item
}

// Report says how an app's cask and the font's stand.
func (b Casks) Report(ctx context.Context, name, app, home string) userfile.Fixed {
	return userfile.Fixed{
		b.Item(ctx, name, b.Has(app, "/Applications", home+"/Applications"),
			"no "+app+" is in /Applications or ~/Applications"),
		b.Item(ctx, Font, b.Has(FontFile, home+"/Library/Fonts", "/Library/Fonts"), FontGone),
	}
}

// Font plans the font's cask where its file is missing.
func (b Casks) Font(ctx context.Context, home string) ([]step.Change, error) {
	c, _, err := b.Cask(ctx, Font, b.Has(FontFile, home+"/Library/Fonts", "/Library/Fonts"), FontGone)
	return c, err
}

// Install applies a plan's changes. A failed cask install is a warning, and the next run tries
// again; where the app's fails, the other changes wait for it.
func (b Casks) Install(ctx context.Context, p step.Plan, app string) error {
	for _, c := range p.Changes {
		if r, ok := c.(change.Run); ok {
			cask := r.Args[len(r.Args)-1]
			b.Out.Say("installing %s", cask)
			if err := change.Apply(ctx, b.M, b.Out, c); err != nil {
				b.Out.Warn("brew install --cask %s failed", cask)
				b.Failed[cask] = true
			}
			continue
		}
		if b.Failed[app] {
			continue
		}
		if err := change.Apply(ctx, b.M, b.Out, c); err != nil {
			return err
		}
	}
	return nil
}
