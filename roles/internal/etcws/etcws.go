// Package etcws names ws's one directory, /etc/ws, root's and 755 on both sides, and the resources
// that write a file in it, root's and 644.
package etcws

import (
	"context"
	"fmt"
	"io/fs"
	pathpkg "path"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
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

// Files returns /etc/ws, then each directory under it down to the file's own, and then the file
// at path, whole, all root's. Everyone may read them, so a run that changes nothing asks for no
// password.
func Files(m effect.Machine, path string, content []byte) []resource.Resource {
	return append([]resource.Resource{Top(m)}, Inside(m, path, content)...)
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

// Write puts content in the file at path, in /etc/ws, outside a step, saying each change to out. A
// file and directories as wanted run no sudo, and say nothing. It returns an error and changes
// nothing where a directory on the way is closed to other users, as the tunnel's is. Making it
// public would open what it keeps.
func Write(ctx context.Context, m effect.Machine, out effect.Out, path string, content []byte) error {
	for d := pathpkg.Dir(path); d != Dir && strings.HasPrefix(d, Dir+"/"); d = pathpkg.Dir(d) {
		info, err := m.Stat(d)
		if err == nil && info.Mode.IsDir() && info.Mode.Perm()&0o005 == 0 {
			return fmt.Errorf("%s is closed to other users: %s lies in it, and ws would open it", d, path)
		}
	}
	var changes []step.Change
	for _, part := range Files(m, path, content) {
		var err error
		if changes, err = resource.Append(ctx, changes, resource.Said{Out: out, Part: part}); err != nil {
			return err
		}
	}
	return resource.Group{}.Apply(ctx, step.Plan{Changes: changes})
}
