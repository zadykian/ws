package host

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/zadykian/ws/effect"
)

// Plain refuses a link at any of Paths, or at a directory between Home and them. The user owns its
// home, so a link there would send the root ws runs as, with its reads, chown, chmod and backup, to
// whatever the link names. It looks as it plans, so a link made while the step applies gets past it
// (docs/security.md).
type Plain struct {
	M     effect.Machine
	Home  string
	Paths []string
}

// Plan plans nothing, and fails on the first link.
func (p Plain) Plan(context.Context) ([]Change, error) {
	return nil, RefuseLinks(p.M, p.Home, p.Paths...)
}

// RefuseLinks fails on the first link at any of paths, or at a directory between home and them,
// for a step that learns its paths as it plans. A path outside home is checked alone.
func RefuseLinks(m effect.Machine, home string, paths ...string) error {
	for _, target := range paths {
		for _, at := range chain(home, target) {
			if info, _ := m.Stat(at); info.Link != "" {
				return fmt.Errorf("%s is a link to %s: replace it with the file itself", at, info.Link)
			}
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
