// Package ghostty installs Ghostty and its font on the Mac, and writes the settings that its config
// reads.
package ghostty

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/brew"
)

const (
	// ghosttyHeader starts ws.ghostty's first line, by which a run knows a file for its own.
	ghosttyHeader = "# Written by ws client macos apply"
	// ghosttyLine is the line of Ghostty's config that reads ws.ghostty.
	ghosttyLine = "config-file = ?ws.ghostty"
)

// Ghostty sets up Ghostty, the Mac's terminal. It installs Ghostty and the font its settings name
// from Homebrew's casks where missing, and writes ws.ghostty, those settings, which Ghostty's
// config reads. It warns where Ghostty can't be installed, or Ghostty's config is not the step's to
// change, so that the run goes on to the tunnel. It stops where it can't write a file. Ghostty then
// updates itself, as ws.ghostty says.
type Ghostty struct {
	Home string
	// Settings is the text of roles/client/macos/ghostty/files/ws.ghostty.
	Settings []byte
	M        effect.Machine
	Out      effect.Out
	// LookPath finds a program on the user's PATH. Nil means exec.LookPath.
	LookPath func(string) (string, error)
	// Now stamps the copy of Ghostty's config. Nil means time.Now.
	Now func() time.Time
	// Real resolves the links of a directory. Nil means filepath.EvalSymlinks.
	Real   func(string) (string, error)
	failed map[string]bool // the casks whose install failed, so the plan skips them for the run
}

// casks is the cask code that Ghostty shares with Zed, built from the step's fields.
func (g *Ghostty) casks() brew.Casks {
	if g.failed == nil {
		g.failed = map[string]bool{}
	}
	return brew.Casks{M: g.M, Out: g.Out, LookPath: g.LookPath, Failed: g.failed, App: "ghostty"}
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

// files plans ws.ghostty beside Ghostty's config, from roles/client/macos/ghostty/files/ws.ghostty
// under the header. It plans the line that reads it at the end of that config, once. The old config
// stays as PATH.TIME~. Ghostty reads a config-file after the file that names it (Config.zig's
// config-file), so ws.ghostty's settings win over the config's own. It warns, and leaves the config
// unchanged, where ws.ghostty is not its own, the config can't be read, or the config is a link that
// lacks the line. It stops where ws.ghostty can't be read.
func (g *Ghostty) files(ctx context.Context) ([]step.Change, error) {
	config := g.conf()
	wsFile := g.file(config)
	ws, err := resource.Append(ctx, nil, wsFile)
	switch {
	case errors.Is(err, file.ErrNotWs):
		g.Out.Warn("%s is not ws client macos apply's: move it away, as the script writes its own", wsFile.Path)
		return nil, nil
	case err != nil:
		return nil, err
	}
	lines := g.configLines(config, wsFile.Path)
	found, link, holds, err := lines.Read(ctx)
	if err != nil {
		g.Out.Warn("cannot read %s, Ghostty's config, so ws.ghostty stays out", config)
		return nil, nil
	}
	changes, stop := lines.Changes(found, link, holds, nil)
	if !errors.Is(stop, file.ErrLinked) {
		stop = nil
	}
	dirs, dl, err := g.dirs(config).Plan()
	if err != nil {
		return nil, err
	}
	return g.fileChanges(ws, dirs, dl, holds, changes, stop), nil
}

// dirs is the directory of Ghostty's config alone. Its parent is ~/Library/Application Support on
// any Mac, so a run plans no more.
func (g *Ghostty) dirs(config string) file.UserDirs {
	return file.UserDirs{{M: g.M, Out: g.Out, Path: filepath.Dir(config), Mode: 0o755}}
}

// file is ws.ghostty, beside Ghostty's config.
func (g *Ghostty) file(config string) file.UserFile {
	return file.UserFile{M: g.M, Out: g.Out, Path: filepath.Dir(config) + "/ws.ghostty", Text: g.want(), Mode: 0o644, Own: ghosttyOwn}
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
