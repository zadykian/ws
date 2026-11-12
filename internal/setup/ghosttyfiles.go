package setup

import (
	"io/fs"

	"github.com/zadykian/ws/step"
)

// fileChanges returns the changes to ws.ghostty, ws, after dirs, the changes to its directory,
// whose link warns once. Unless has, they also put the line that reads ws.ghostty at the end of
// Ghostty's config, whose text is text.
func (g *Ghostty) fileChanges(config string, text []byte, ws, dirs []step.Change, link dirLink, has bool) []step.Change {
	cs := dirs
	differs := false
	if len(ws) > 0 {
		_, differs = ws[0].(Write)
	}
	// A linked directory, as GNU stow's folding makes, gets the warning that ssh gives for
	// ~/.ssh/config.d.
	if !has || differs {
		link.warn(g.Out, 1, "ws.ghostty and the line that reads it go there")
	}
	cs = append(cs, ws...)
	if has {
		return cs
	}
	info, err := g.M.Stat(config)
	if err == nil && info.Link != "" {
		g.Out.Warn("%s is a link to %s: add '%s' at the end of that file", config, info.Link, ghosttyLine)
		return cs
	}
	mode := fs.FileMode(0o644)
	if err == nil {
		mode = info.Mode.Perm()
	}
	after := append([]byte(nil), text...)
	if len(after) > 0 && after[len(after)-1] != '\n' {
		after = append(after, '\n')
	}
	after = append(after, ghosttyLine+"\n"...)
	return append(cs, Write{config, text, after, err == nil, mode})
}
