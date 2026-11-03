package helix

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// plain refuses a link at any of its paths. The user owns ~/.config, so a link there would send
// the root ws runs as, with its chown, chmod and backup, to whatever the link names.
type plain struct {
	m     effect.Machine
	paths []string
}

// Plan fails at the first link, before any resource of the step changes anything.
func (p plain) Plan(context.Context) ([]host.Change, error) {
	for _, path := range p.paths {
		if info, _ := p.m.Stat(path); info.Link != "" {
			return nil, fmt.Errorf("%s is a link to %s: replace it with the file itself", path, info.Link)
		}
	}
	return nil, nil
}
