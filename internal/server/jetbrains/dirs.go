package jetbrains

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/zadykian/ws/host"
)

// Dirs is a directory under the user's home, with the missing directories above it. The user makes
// those it adds, with the mode 0755 that u+rwx gives under the usual umask. A directory that is
// there keeps its mode and owner, as Gateway may have made it.
type Dirs struct {
	S    Site
	Path string
}

type makeDir struct {
	s    Site
	path string
}

func (c makeDir) String() string {
	return "create directory " + c.path + ", mode 755"
}

func (c makeDir) Apply(ctx context.Context) error {
	_, err := c.s.AsUser(ctx, "mkdir", "-m", "0755", "--", c.path)
	return err
}

// Plan makes a change for each directory that is missing, the topmost first.
func (d Dirs) Plan(context.Context) ([]host.Change, error) {
	rel, ok := strings.CutPrefix(d.Path, d.S.Home+"/")
	if !ok {
		return nil, fmt.Errorf("%s lies outside the home %s", d.Path, d.S.Home)
	}
	var changes []host.Change
	dir := d.S.Home
	for _, part := range strings.Split(rel, "/") {
		dir = path.Join(dir, part)
		if len(changes) == 0 {
			_, err := d.S.M.Stat(dir)
			if err == nil {
				continue
			}
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		}
		changes = append(changes, makeDir{d.S, dir})
	}
	return changes, nil
}
