package setup

import (
	"context"
	"errors"
	"os/exec"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

const (
	// caskFont is the cask of the font that Ghostty's and Zed's settings name, and caskFontFile the
	// file of its regular style.
	caskFont     = "font-jetbrains-mono-nerd-font"
	caskFontFile = "JetBrainsMonoNerdFontMono-Regular.ttf"
)

// brewCasks plans and installs an app's Homebrew cask and the font's, for the steps that set up an
// app. A step builds it per call from its own fields, and the failed map is shared with the step.
type brewCasks struct {
	m        effect.Machine
	out      effect.Out
	lookPath func(string) (string, error)
	failed   map[string]bool // the casks whose install failed, so the plan skips them for the run
}

// has reports whether the first of the files is there, in /Applications or ~/Applications for the
// app, where its cask puts it, or in ~/Library/Fonts or /Library/Fonts for the font.
func (b brewCasks) has(name string, dirs ...string) bool {
	for _, d := range dirs {
		if _, err := b.m.Stat(d + "/" + name); err == nil {
			return true
		}
	}
	return false
}

// cask plans Homebrew's cask where there is none of its files, and says whether they are there,
// or would be after the plan. Where Homebrew has the cask but its files are gone, it warns instead,
// as gateway does: brew install would upgrade the cask. Without Homebrew it stops.
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
		return []step.Change{Run{Name: "brew", Args: []string{"install", "--cask", name}}}, true, nil
	}
	b.out.Warn("Homebrew has the cask %s, but %s: brew reinstall --cask %s puts it back",
		name, missing, name)
	return nil, false, nil
}

// font plans the font's cask where its file is missing.
func (b brewCasks) font(ctx context.Context, home string) ([]step.Change, error) {
	c, _, err := b.cask(ctx, caskFont, b.has(caskFontFile, home+"/Library/Fonts", "/Library/Fonts"),
		"no "+caskFontFile+" is in ~/Library/Fonts or /Library/Fonts")
	return c, err
}

// install applies a plan's changes. A failed cask install is a warning, and the next run tries
// again; where the app's fails, the other changes wait for it.
func (b brewCasks) install(ctx context.Context, p step.Plan, app string) error {
	for _, c := range p.Changes {
		if r, ok := c.(Run); ok {
			cask := r.Args[len(r.Args)-1]
			b.out.Say("installing %s", cask)
			if err := Apply(ctx, b.m, b.out, c); err != nil {
				b.out.Warn("brew install --cask %s failed", cask)
				b.failed[cask] = true
			}
			continue
		}
		if b.failed[app] {
			continue
		}
		if err := Apply(ctx, b.m, b.out, c); err != nil {
			return err
		}
	}
	return nil
}
