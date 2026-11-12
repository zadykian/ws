package setup

import (
	"context"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// configLines is the line that reads ws.ghostty, at the end of Ghostty's config.
func (g *Ghostty) configLines(config, ws string) userLines {
	return userLines{
		M: g.M, Path: config, Lines: ghosttyLine + "\n", Shown: ghosttyLine, Mode: 0o644, Now: g.Now,
		Has: func(_ context.Context, _ effect.Machine, _ string, text []byte) (bool, error) {
			return ghosttyIncludes(config, string(text), ws, g.Home, g.resolve()), nil
		},
	}
}

// fileChanges returns the changes to ws.ghostty, ws, after dirs, the changes to its directory,
// whose link warns once, and then lines, the changes to Ghostty's config. A stop of the config's,
// such as a link that lacks the line, warns, and the config stays unchanged.
func (g *Ghostty) fileChanges(ws, dirs []step.Change, link dirLink, holds bool, lines []step.Change, stop error) []step.Change {
	cs := dirs
	differs := false
	if len(ws) > 0 {
		_, differs = ws[0].(Write)
	}
	// A linked directory, as GNU stow's folding makes, gets the warning that ssh gives for
	// ~/.ssh/config.d.
	if !holds || differs {
		link.warn(g.Out, 1, "ws.ghostty and the line that reads it go there")
	}
	if stop != nil {
		g.Out.Warn("%s", stop)
	}
	return append(append(cs, ws...), lines...)
}
