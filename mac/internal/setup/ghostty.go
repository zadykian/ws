package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

const (
	// ghosttyFont is the cask of the font that ws.ghostty names, and ghosttyFontFile the file of its
	// regular style.
	ghosttyFont     = "font-jetbrains-mono-nerd-font"
	ghosttyFontFile = "JetBrainsMonoNerdFontMono-Regular.ttf"
	// ghosttyHeader starts ws.ghostty's first line, by which a run knows a file for its own.
	ghosttyHeader = "# Written by ws's mac/setup.sh"
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

// Plan plans the app, then the font where the app is there or comes, and the files.
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
	files, err := g.files()
	if err != nil {
		return step.Plan{}, err
	}
	return step.Plan{Changes: append(append(app, font...), files...)}, nil
}

// Apply installs the casks. A failed install is a warning, and the next run tries again; where
// the app's fails, the files wait for it.
func (g *Ghostty) Apply(ctx context.Context, p step.Plan) error {
	if g.failed == nil {
		g.failed = map[string]bool{}
	}
	for _, c := range p.Changes {
		if r, ok := c.(Run); ok {
			cask := r.Args[len(r.Args)-1]
			g.Out.Say("installing %s", cask)
			if err := Apply(ctx, g.M, g.Out, c); err != nil {
				g.Out.Warn("brew install --cask %s failed", cask)
				g.failed[cask] = true
			}
			continue
		}
		if g.failed["ghostty"] {
			continue
		}
		if err := Apply(ctx, g.M, g.Out, c); err != nil {
			return err
		}
	}
	return nil
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
func (g *Ghostty) files() ([]step.Change, error) {
	config := g.conf()
	dir := filepath.Dir(config)
	file := dir + "/ws.ghostty"
	_, err := g.M.Stat(file)
	exists := err == nil
	var old []byte
	if exists {
		old, _ = g.M.ReadFile(file)
		if first, _, _ := strings.Cut(string(old), "\n"); !strings.HasPrefix(first, ghosttyHeader) {
			g.Out.Warn("%s is not mac/setup.sh's: move it away, as the script writes its own", file)
			return nil, nil
		}
	}
	text, err := g.M.ReadFile(config)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		g.Out.Warn("cannot read %s, Ghostty's config, so ws.ghostty stays out", config)
		return nil, nil
	}
	settings, err := g.M.ReadFile(g.Settings)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", g.Settings, err)
	}
	want := append([]byte(ghosttyHeader+": its next run replaces changes made here.\n"), settings...)
	resolve := g.Real
	if resolve == nil {
		resolve = filepath.EvalSymlinks
	}
	has := ghosttyIncludes(config, string(text), file, g.Home, resolve)
	return g.fileChanges(config, text, file, old, want, has), nil
}
