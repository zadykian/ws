package launchd

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Reread plans a SIGHUP of each daemon that launchd has loaded, so that each reads routes.d and the
// other servers' files again. A daemon skips a network that another server's routes.d holds, so a
// change to one reaches them all.
func Reread(ctx context.Context, m effect.Machine, out effect.Out) ([]step.Change, error) {
	names, err := Installed(m)
	if err != nil {
		return nil, err
	}
	var cs []step.Change
	for _, name := range names {
		label := For(name).Label
		if _, err := m.Run(ctx, "launchctl", "print", "system/"+label); err == nil {
			cs = append(cs, Launchctl{M: m, Out: out,
				Args: []string{"kill", "SIGHUP", "system/" + label}, Label: label, Routes: true})
		}
	}
	return cs, nil
}
