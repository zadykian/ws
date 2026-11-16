package ghostty

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
)

// configLines is the line that reads /etc/ws/ghostty, at the end of Ghostty's config.
func (g *Ghostty) configLines(config string) file.UserLines {
	return file.UserLines{
		M: g.M, Out: g.Out, Path: config, Lines: ghosttyLine + "\n", Shown: ghosttyLine, Mode: 0o644, Now: g.Now,
		Has: func(_ context.Context, _ effect.Machine, _ string, text []byte) (bool, error) {
			return ghosttyIncludes(config, string(text), settingsFile, g.Home, g.resolve()), nil
		},
	}
}

// fileChanges returns the changes to /etc/ws and its settings, then dirs, those to Ghostty's
// directory, whose link warns once, and then lines, those to Ghostty's config. A stop of the
// config's, such as a link that lacks the line, warns. The config stays unchanged.
func (g *Ghostty) fileChanges(settings, dirs []step.Change, link file.UserDirLink, holds bool, lines []step.Change, stop error) []step.Change {
	// A linked directory, as GNU stow's folding makes, gets a warning.
	if !holds {
		link.Warn(g.Out, 1, "the line that reads "+settingsFile+" goes there")
	}
	if stop != nil {
		g.Out.Warn("%s", stop)
	}
	return append(append(settings, dirs...), lines...)
}
