// Package etcws names ws's one directory, /etc/ws, root's and 755 on both sides, and the resources
// that write a file in it, root's and 644.
package etcws

import (
	"io/fs"

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

// Files returns the directory and then the file at path, whole, root's. Everyone may read them, so
// a run that changes nothing asks for no password.
func Files(m effect.Machine, path string, content []byte) []resource.Resource {
	return []resource.Resource{
		Top(m),
		file.Public{M: m, Path: path, Content: content, Mode: FileMode},
	}
}
