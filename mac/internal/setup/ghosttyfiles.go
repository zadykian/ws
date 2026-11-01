package setup

import (
	"bytes"
	"io/fs"
	"path/filepath"

	"github.com/zadykian/ws/mac/internal/step"
)

// fileChanges returns the changes that put want in ws.ghostty, whose text is old where it exists,
// and the line that reads it at the end of Ghostty's config, whose text is text, unless has.
func (g *Ghostty) fileChanges(config string, text []byte, file string, old, want []byte, has bool) []step.Change {
	dir := filepath.Dir(file)
	var cs []step.Change
	dirInfo, dirErr := g.M.Stat(dir)
	fileInfo, fileErr := g.M.Stat(file)
	differs := fileErr != nil || !bytes.Equal(old, want)
	// A linked directory, as GNU stow's folding makes, gets the warning that ssh gives for
	// ~/.ssh/config.d.
	if dirErr == nil && dirInfo.Link != "" && (!has || differs) {
		g.Out.Warn("%s is a link to %s: ws.ghostty and the line that reads it go there", dir, dirInfo.Link)
	}
	if dirErr != nil {
		cs = append(cs, MakeDir{dir, 0o755})
	}
	switch {
	case differs:
		cs = append(cs, Write{file, old, want, fileErr == nil, 0o644})
	case fileInfo.Mode.Perm() != 0o644:
		cs = append(cs, SetMode{file, 0o644})
	}
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
