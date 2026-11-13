package ghostty

import (
	"context"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/brew"
)

// Step is ghostty as a step of resources. A failed cask install is a warning, and the next run
// tries again; where the app's fails, the files wait for it.
func (g *Ghostty) Step() resource.Lazy {
	return resource.Lazy{Title: "ghostty", Parts: g.parts}
}

// parts is the app's cask, the font's, then the files. The font and the files plan where the app
// is there or comes. A part reads the Mac when it plans or reports, so that a report warns of
// nothing.
func (g *Ghostty) parts(context.Context) ([]resource.Resource, error) {
	b := g.casks()
	app := &brew.App{Cask: b.Resource("ghostty", b.Has("Ghostty.app", "/Applications", g.Home+"/Applications"),
		"no Ghostty.app is in /Applications or ~/Applications")}
	return []resource.Resource{app, app.Then(b.FontResource(g.Home)), app.Then(ghosttyFiles{g})}, nil
}

// ghosttyFiles is ws.ghostty and the line that reads it.
type ghosttyFiles struct{ g *Ghostty }

// Plan returns the changes of the files, which warn where they can't be planned.
func (f ghosttyFiles) Plan(ctx context.Context) ([]resource.Change, error) {
	changes, err := f.g.files(ctx)
	return resource.Planned{Changes: changes, Err: err}.Plan(ctx)
}

// Report says how ws.ghostty and the line that reads it stand. A config that is a link lacking the
// line stands against ws: a run only warns of it.
func (f ghosttyFiles) Report(ctx context.Context) ([]step.Item, error) {
	config := f.g.conf()
	wsFile := f.g.file(config)
	return file.Gather(ctx, f.g.dirs(config), wsFile, f.g.configLines(config, wsFile.Path))
}
