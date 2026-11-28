// Package etcws names ws's one directory, /etc/ws, root's and 755 on both sides. Its resources
// make the directory and a file in it, root's and 644.
package etcws

import (
	"io/fs"
	pathpkg "path"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
)

// Dir is the directory, DirMode its mode and FileMode a file's.
const (
	Dir                  = "/etc/ws"
	DirMode  fs.FileMode = 0o755
	FileMode fs.FileMode = 0o644
)

// Top is /etc/ws itself, made where missing. Everyone may read it, so a run that changes nothing
// asks for no password.
func Top(m effect.Machine) file.PublicDir {
	return file.PublicDir{M: m, Path: Dir, Mode: DirMode}
}

// Inside returns each directory under /etc/ws down to the file's own, and then the file at path, for
// a plan that holds /etc/ws already. A plan holds it once, as a second make fails.
func Inside(m effect.Machine, path string, content []byte) []resource.Resource {
	var dirs []string
	for d := pathpkg.Dir(path); d != Dir && strings.HasPrefix(d, Dir+"/"); d = pathpkg.Dir(d) {
		dirs = append(dirs, d)
	}
	var parts []resource.Resource
	for _, d := range slices.Backward(dirs) {
		parts = append(parts, file.PublicDir{M: m, Path: d, Mode: DirMode})
	}
	return append(parts, file.Public{M: m, Path: path, Content: content, Mode: FileMode})
}

// Holds tells whether p lies in /etc/ws, where root owns each file. A file elsewhere is the user's.
func Holds(p string) bool {
	return strings.HasPrefix(pathpkg.Clean(p), Dir+"/")
}
