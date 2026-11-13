package host

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// makeParents makes each directory between home and target that is missing, top first, as the
// owner's at 0755, and leaves an existing one alone. Without a home it does nothing.
func makeParents(m effect.Machine, home, target string, uid, gid int) error {
	if home == "" {
		return nil
	}
	dirs := chain(home, target)
	for _, at := range dirs[:len(dirs)-1] {
		_, err := m.Stat(at)
		if errors.Is(err, fs.ErrNotExist) {
			err = errors.Join(m.Mkdir(at, 0o755), m.Chown(at, uid, gid), m.Chmod(at, 0o755))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// guardHome fails unless target lies strictly below home by a local path, and fails on the first
// link at target or at a directory between home and it.
func guardHome(m effect.Machine, home, target string) error {
	rel, ok := strings.CutPrefix(target, home+"/")
	if !ok || !filepath.IsLocal(rel) {
		return fmt.Errorf("%s is not a path below the home %s", target, home)
	}
	for _, at := range chain(home, target) {
		info, err := m.Stat(at)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("look for a link at %s: %w", at, err)
		}
		if info.Link != "" {
			return fmt.Errorf("%s is a link to %s: replace it with the file itself", at, info.Link)
		}
	}
	return nil
}

// chain returns each directory below home on the way to target, then target.
func chain(home, target string) []string {
	rel, ok := strings.CutPrefix(target, home+"/")
	if !ok {
		return []string{target}
	}
	var list []string
	at := home
	for part := range strings.SplitSeq(rel, "/") {
		at = path.Join(at, part)
		list = append(list, at)
	}
	return list
}
