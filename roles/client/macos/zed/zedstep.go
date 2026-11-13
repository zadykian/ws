package zed

import (
	"context"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/brew"
)

// Step is Zed as a step of resources. A failed cask install is a warning, and the next run
// tries again; where the app's fails, the files wait for it.
func (z *Zed) Step() resource.Lazy {
	return resource.Lazy{Title: "zed", Parts: z.parts}
}

// parts is the app's cask, the font's, then the files. The font and the files plan where the app
// is there or comes. A part reads the Mac when it plans or reports, so that a report warns of
// nothing.
func (z *Zed) parts(context.Context) ([]resource.Resource, error) {
	b := z.casks()
	app := &brew.App{Cask: b.Resource("zed", b.Has("Zed.app", "/Applications", z.Home+"/Applications"),
		"no Zed.app is in /Applications or ~/Applications")}
	return []resource.Resource{app, app.Then(b.FontResource(z.Home)), app.Then(zedFiles{z})}, nil
}

// zedFiles is the directories and the files in ~/.config/zed.
type zedFiles struct{ z *Zed }

// Plan returns the changes of the files, which warn where they can't be planned.
func (f zedFiles) Plan(ctx context.Context) ([]resource.Change, error) {
	changes, err := f.z.files(ctx)
	return resource.Planned{Changes: changes, Err: err}.Plan(ctx)
}

// Report says how the files stand. settings.json is Zed's own, which ws writes once and never
// reads, so a report only finds it.
func (f zedFiles) Report(ctx context.Context) ([]step.Item, error) {
	files := f.z.whole()
	return file.Gather(ctx, f.z.dirs(), files[0], files[1],
		file.UserPath{M: f.z.M, Kind: step.FileItem, Path: f.z.Home + "/.config/zed/settings.json"})
}
