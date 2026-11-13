package setup

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
)

const (
	// caskFont is the cask of the font that Ghostty's and Zed's settings name, and caskFontFile the
	// file of its regular style.
	caskFont     = "font-jetbrains-mono-nerd-font"
	caskFontFile = "JetBrainsMonoNerdFontMono-Regular.ttf"
	fontGone     = "no " + caskFontFile + " is in ~/Library/Fonts or /Library/Fonts"
)

// brewCasks plans and installs an app's Homebrew cask and the font's, for the steps that set up an
// app. A step builds it per call from its own fields, and the failed map is shared with the step.
type brewCasks struct {
	m        effect.Machine
	out      effect.Out
	lookPath func(string) (string, error)
	failed   map[string]bool // the casks whose install failed, so the plan skips them for the run
}

// has reports whether the first of the files is there. An app's cask puts it in /Applications or
// ~/Applications, and a font's in ~/Library/Fonts or /Library/Fonts.
func (b brewCasks) has(name string, dirs ...string) bool {
	for _, d := range dirs {
		if _, err := b.m.Stat(d + "/" + name); err == nil {
			return true
		}
	}
	return false
}

// cask plans Homebrew's cask where there is none of its files, and says whether they are there,
// or would be after the plan. Where Homebrew has the cask but its files are gone, it warns instead:
// brew install would upgrade the cask. Without Homebrew it stops.
func (b brewCasks) cask(ctx context.Context, name string, there bool, missing string) ([]step.Change, bool, error) {
	if there {
		return nil, true, nil
	}
	if b.failed[name] {
		return nil, false, nil
	}
	look := b.lookPath
	if look == nil {
		look = exec.LookPath
	}
	if _, err := look("brew"); err != nil {
		return nil, false, errors.New("install Homebrew first: https://brew.sh")
	}
	if _, err := b.m.Run(ctx, "brew", "list", "--cask", name); err != nil {
		return []step.Change{change.Run{Name: "brew", Args: []string{"install", "--cask", name}}}, true, nil
	}
	b.out.Warn("%s", caskGone(name, missing))
	return nil, false, nil
}

func caskGone(name, missing string) string {
	return fmt.Sprintf("Homebrew has the cask %s, but %s: brew reinstall --cask %s puts it back",
		name, missing, name)
}

// item says how the cask stands, as cask reads it. brew is asked of one cask, and only where its
// files are gone: a report covers what ws manages, never all that Homebrew holds.
func (b brewCasks) item(ctx context.Context, name string, there bool, missing string) step.Item {
	item := step.Item{Kind: step.PackageItem, Name: name, Want: stateInstalled, Found: stateInstalled}
	if there {
		return item
	}
	item.Found = stateMissing
	if _, err := b.m.Run(ctx, "brew", "list", "--cask", name); err == nil {
		item.Conflict = caskGone(name, missing)
	}
	return item
}

// report says how an app's cask and the font's stand.
func (b brewCasks) report(ctx context.Context, name, app, home string) fixed {
	return fixed{
		b.item(ctx, name, b.has(app, "/Applications", home+"/Applications"),
			"no "+app+" is in /Applications or ~/Applications"),
		b.item(ctx, caskFont, b.has(caskFontFile, home+"/Library/Fonts", "/Library/Fonts"), fontGone),
	}
}

// font plans the font's cask where its file is missing.
func (b brewCasks) font(ctx context.Context, home string) ([]step.Change, error) {
	c, _, err := b.cask(ctx, caskFont, b.has(caskFontFile, home+"/Library/Fonts", "/Library/Fonts"), fontGone)
	return c, err
}

// install applies a plan's changes. A failed cask install is a warning, and the next run tries
// again; where the app's fails, the other changes wait for it.
func (b brewCasks) install(ctx context.Context, p step.Plan, app string) error {
	for _, c := range p.Changes {
		if r, ok := c.(change.Run); ok {
			cask := r.Args[len(r.Args)-1]
			b.out.Say("installing %s", cask)
			if err := change.Apply(ctx, b.m, b.out, c); err != nil {
				b.out.Warn("brew install --cask %s failed", cask)
				b.failed[cask] = true
			}
			continue
		}
		if b.failed[app] {
			continue
		}
		if err := change.Apply(ctx, b.m, b.out, c); err != nil {
			return err
		}
	}
	return nil
}
