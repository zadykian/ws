// Package brew plans and installs the Homebrew casks of an app and its font, for the roles that
// set up an app.
package brew

import (
	"context"
	"fmt"
	"github.com/zadykian/ws/engine/core/resource/file"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
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
	App      string          // the app's cask: the changes after it wait where its install fails
}

// Install is a cask's install. A failure is a warning, not an error, and the next run tries again.
type Install struct {
	resource.Command
	Out    effect.Out
	Failed map[string]bool
	Name   string
	Holds  bool
}

// Apply says the install, runs it, and on a failure warns and marks the cask failed for the run.
func (i Install) Apply(ctx context.Context) error {
	i.Out.Say("installing %s", i.Name)
	err := i.Command.Apply(ctx)
	if err != nil {
		i.Out.Warn("brew install --cask %s failed", i.Name)
		i.Failed[i.Name] = true
	}
	return err
}

// Warned is Holds: the changes after the app's install wait for it.
func (i Install) Warned() bool { return i.Holds }

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
// or would be after the plan. See Cask.Plan.
func (b Casks) Cask(ctx context.Context, name string, there bool, missing string) ([]step.Change, bool, error) {
	changes, ok, err := b.Resource(name, there, missing).plan(ctx)
	return asSteps(changes), ok, err
}

func asSteps(changes []resource.Change) []step.Change {
	cs := make([]step.Change, len(changes))
	for i, c := range changes {
		cs[i] = c
	}
	return cs
}

// Resource is the cask as a resource. The app's cask holds the changes after it.
func (b Casks) Resource(name string, there bool, missing string) Cask {
	return Cask{M: b.M, Out: b.Out, LookPath: b.LookPath, Failed: b.Failed, Name: name, There: there,
		Missing: missing, Holds: name == b.App}
}

// CaskGone says why a cask that Homebrew has is missing from the Mac.
func CaskGone(name, missing string) string {
	return fmt.Sprintf("Homebrew has the cask %s, but %s: brew reinstall --cask %s puts it back",
		name, missing, name)
}

// Item says how the cask stands. See Cask.Report.
func (b Casks) Item(ctx context.Context, name string, there bool, missing string) step.Item {
	return b.Resource(name, there, missing).item(ctx)
}

// Report says how an app's cask and the font's stand.
func (b Casks) Report(ctx context.Context, name, app, home string) file.Fixed {
	return file.Fixed{
		b.Item(ctx, name, b.Has(app, "/Applications", home+"/Applications"),
			"no "+app+" is in /Applications or ~/Applications"),
		b.Item(ctx, Font, b.Has(FontFile, home+"/Library/Fonts", "/Library/Fonts"), FontGone),
	}
}

// FontResource is the font's cask, which holds nothing back.
func (b Casks) FontResource(home string) Cask {
	return b.Resource(Font, b.Has(FontFile, home+"/Library/Fonts", "/Library/Fonts"), FontGone)
}

// Font plans the font's cask where its file is missing.
func (b Casks) Font(ctx context.Context, home string) ([]step.Change, error) {
	changes, err := b.FontResource(home).Plan(ctx)
	return asSteps(changes), err
}

// Install applies a plan's changes. A failed cask install is a warning, and the next run tries
// again; where the app's fails, the other changes wait for it.
func (b Casks) Install(ctx context.Context, p step.Plan, app string) error {
	for _, c := range p.Changes {
		if i, ok := c.(Install); ok {
			_ = i.Apply(ctx)
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
