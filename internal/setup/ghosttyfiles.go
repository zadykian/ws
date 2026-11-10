package setup

import (
	"io/fs"
	"path/filepath"

	"github.com/zadykian/ws/step"
)

// fileChanges returns the changes that make the directory of ws.ghostty, file, where missing, then
// ws, the changes to that file. Unless has, they also put the line that reads ws.ghostty at the end
// of Ghostty's config, whose text is text.
func (g *Ghostty) fileChanges(config string, text []byte, file string, ws []step.Change, has bool) []step.Change {
	dir := filepath.Dir(file)
	var cs []step.Change
	dirInfo, dirErr := g.M.Stat(dir)
	differs := false
	if len(ws) > 0 {
		_, differs = ws[0].(Write)
	}
	// A linked directory, as GNU stow's folding makes, gets the warning that ssh gives for
	// ~/.ssh/config.d.
	if dirErr == nil && dirInfo.Link != "" && (!has || differs) {
		g.Out.Warn("%s is a link to %s: ws.ghostty and the line that reads it go there", dir, dirInfo.Link)
	}
	if dirErr != nil {
		cs = append(cs, MakeDir{dir, 0o755})
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
