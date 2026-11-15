// Package ghostty installs Ghostty and its font on the Mac, and writes the settings that its config
// reads.
package ghostty

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/brew"
	"github.com/zadykian/ws/roles/internal/etcws"
)

const (
	// settingsFile is ws's settings, root's, which Ghostty's config reads.
	settingsFile = etcws.Dir + "/ghostty"
	// ghosttyLine is the line of Ghostty's config that reads settingsFile.
	ghosttyLine = "config-file = ?" + settingsFile
)

// Ghostty sets up Ghostty, the Mac's terminal. It installs Ghostty and the font its settings name
// from Homebrew's casks where missing, and writes /etc/ws/ghostty, those settings, which Ghostty's
// config reads. It warns where Ghostty can't be installed, or Ghostty's config is not the step's to
// change, so that the run goes on to the tunnel. It stops where it can't write a file. Ghostty then
// updates itself, as /etc/ws/ghostty says.
type Ghostty struct {
	Home string
	// Settings is the text of roles/client/macos/ghostty/files/ws.ghostty, which /etc/ws/ghostty holds.
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

// files plans /etc/ws/ghostty, from roles/client/macos/ghostty/files/ws.ghostty under the header, as
// root's, and replaced whole. It plans the line that reads it at the end of Ghostty's config, once.
// The old config stays as PATH.TIME~. Ghostty reads a config-file after the file that names it
// (Config.zig's config-file), so /etc/ws/ghostty's settings win over the config's own. It warns, and
// leaves the config unchanged, where the config can't be read, or the config is a link that lacks
// the line.
func (g *Ghostty) files(ctx context.Context) ([]step.Change, error) {
	config := g.conf()
	var settings []step.Change
	for _, part := range g.settings() {
		var err error
		if settings, err = resource.Append(ctx, settings, part); err != nil {
			return nil, err
		}
	}
	lines := g.configLines(config)
	found, link, holds, err := lines.Read(ctx)
	if err != nil {
		g.Out.Warn("cannot read %s, Ghostty's config, so %s stays out", config, settingsFile)
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
	return g.fileChanges(settings, dirs, dl, holds, changes, stop), nil
}

// dirs is the directory of Ghostty's config alone. Its parent is ~/Library/Application Support on
// any Mac, so a run plans no more.
func (g *Ghostty) dirs(config string) file.UserDirs {
	return file.UserDirs{{M: g.M, Out: g.Out, Path: filepath.Dir(config), Mode: 0o755}}
}

// settings is /etc/ws and ws's settings in it, each saying its change as it applies.
func (g *Ghostty) settings() []resource.Said {
	var parts []resource.Said
	for _, r := range etcws.Files(g.M, settingsFile, g.want()) {
		parts = append(parts, resource.Said{Out: g.Out, Part: r})
	}
	return parts
}

// want is what /etc/ws/ghostty holds.
func (g *Ghostty) want() []byte {
	return append([]byte(file.Header("#")+"\n"), g.Settings...)
}

func (g *Ghostty) resolve() func(string) (string, error) {
	if g.Real != nil {
		return g.Real
	}
	return filepath.EvalSymlinks
}
