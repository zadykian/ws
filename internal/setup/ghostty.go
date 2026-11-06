package setup

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

const (
	// ghosttyHeader starts ws.ghostty's first line, by which a run knows a file for its own.
	ghosttyHeader = "# Written by ws mac apply"
	// ghosttyLine is the line of Ghostty's config that reads ws.ghostty.
	ghosttyLine = "config-file = ?ws.ghostty"
)

// Ghostty sets up Ghostty, the Mac's terminal: Ghostty and the font its settings name, from
// Homebrew's casks where missing, and ws.ghostty, those settings, which Ghostty's config reads.
// It warns where Ghostty can't be installed, or Ghostty's config is not the step's to change, so
// that the run goes on to the tunnel. It stops where it can't write a file. Ghostty then updates
// itself, as ws.ghostty says.
type Ghostty struct {
	Home string
	// Settings is the text of mac/ws.ghostty.
	Settings []byte
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

// casks is the cask code that Ghostty shares with Zed, built from the step's fields.
func (g *Ghostty) casks() brewCasks {
	return brewCasks{m: g.M, out: g.Out, lookPath: g.LookPath, failed: g.failed}
}

// Plan plans the app, then the font where the app is there or comes, and the files.
func (g *Ghostty) Plan(ctx context.Context) (step.Plan, error) {
	b := g.casks()
	app, ok, err := b.cask(ctx, "ghostty", b.has("Ghostty.app", "/Applications", g.Home+"/Applications"),
		"no Ghostty.app is in /Applications or ~/Applications")
	if err != nil || !ok {
		return step.Plan{}, err
	}
	font, err := b.font(ctx, g.Home)
	if err != nil {
		return step.Plan{}, err
	}
	return step.Plan{Changes: append(append(app, font...), g.files()...)}, nil
}

// Apply installs the casks. A failed install is a warning, and the next run tries again; where
// the app's fails, the files wait for it.
func (g *Ghostty) Apply(ctx context.Context, p step.Plan) error {
	if g.failed == nil {
		g.failed = map[string]bool{}
	}
	return g.casks().install(ctx, p, "ghostty")
}

// conf returns the config file that Ghostty's menu opens: the first of ghosttyConfigs that holds
// anything, else the first that exists, else the first.
func (g *Ghostty) conf() string {
	configs := ghosttyConfigs(g.Home)
	found := ""
	for _, c := range configs {
		info, err := g.M.Stat(c)
		if err != nil || !info.Mode.IsRegular() {
			continue
		}
		if text, err := g.M.ReadFile(c); err != nil || len(text) > 0 {
			return c
		}
		if found == "" {
			found = c
		}
	}
	if found == "" {
		return configs[0]
	}
	return found
}

// files plans ws.ghostty beside Ghostty's config, mac/ws.ghostty under the header, and the line
// that reads it at the end of that config, once. Ghostty reads a config-file after the file that
// names it (Config.zig's config-file), so ws.ghostty's settings win over the config's own. It
// warns, and leaves the config unchanged, where ws.ghostty is not its own, the config can't be
// read, or the config is a link that lacks the line.
func (g *Ghostty) files() []step.Change {
	config := g.conf()
	dir := filepath.Dir(config)
	file := dir + "/ws.ghostty"
	_, err := g.M.Stat(file)
	exists := err == nil
	var old []byte
	if exists {
		old, _ = g.M.ReadFile(file)
		if !ghosttyOwn(old) {
			g.Out.Warn("%s is not ws mac apply's: move it away, as the script writes its own", file)
			return nil
		}
	}
	text, err := g.M.ReadFile(config)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		g.Out.Warn("cannot read %s, Ghostty's config, so ws.ghostty stays out", config)
		return nil
	}
	has := ghosttyIncludes(config, string(text), file, g.Home, g.resolve())
	return g.fileChanges(config, text, file, old, g.want(), has)
}

// ghosttyOwn says whether ws.ghostty's first line is ws's.
func ghosttyOwn(have []byte) bool {
	first, _, _ := strings.Cut(string(have), "\n")
	return strings.HasPrefix(first, ghosttyHeader)
}

// want is what ws.ghostty holds.
func (g *Ghostty) want() []byte {
	return append([]byte(ghosttyHeader+": its next run replaces changes made here.\n"), g.Settings...)
}

func (g *Ghostty) resolve() func(string) (string, error) {
	if g.Real != nil {
		return g.Real
	}
	return filepath.EvalSymlinks
}

// Report says how the casks, ws.ghostty and the line that reads it stand. A config that is a link
// lacking the line stands against ws: a run only warns of it.
func (g *Ghostty) Report(ctx context.Context) ([]step.Item, error) {
	config := g.conf()
	dir := filepath.Dir(config)
	file := dir + "/ws.ghostty"
	text, err := g.M.ReadFile(config)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	line := lineItem(config, ghosttyLine, err == nil, ghosttyIncludes(config, string(text), file, g.Home, g.resolve()))
	if info, err := g.M.Stat(config); err == nil && info.Link != "" && line.Found != line.Want {
		line.Conflict = "a link to " + info.Link + ": add the line at the end of that file"
	}
	return gather(ctx, g.casks().report(ctx, "ghostty", "Ghostty.app", g.Home),
		userPath{M: g.M, Kind: step.DirItem, Path: dir},
		userFile{M: g.M, Path: file, Text: g.want(), Mode: 0o644, Own: ghosttyOwn},
		fixed{line})
}
