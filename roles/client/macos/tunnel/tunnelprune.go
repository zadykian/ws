package tunnel

import (
	"context"
	"slices"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
)

// gone returns the servers whose daemon has files on the Mac and which no longer have a directory
// beside mac.conf. It returns none unless mac.conf's servers list this one: that shows the listing
// is this Mac's own, so a stray WS_CONF removes no live daemon.
func (t *Tunnel) gone(ctx context.Context) ([]string, error) {
	servers, err := conf.Servers(t.M, t.MacConf)
	if err != nil || !slices.Contains(servers, t.Server) {
		return nil, err
	}
	found, err := launchd.Found(ctx, t.M)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(found, func(n string) bool { return !conf.IsName(n) || slices.Contains(servers, n) }), nil
}

// prune plans, for each server that is gone, a bootout where launchd has its daemon loaded, then
// the removal of its plist, directory, state file and log.
func (t *Tunnel) prune(ctx context.Context, gone []string) ([]step.Change, error) {
	var cs []step.Change
	for _, name := range gone {
		d := launchd.For(name)
		if _, err := t.M.Run(ctx, "launchctl", "print", "system/"+d.Label); err == nil {
			cs = append(cs, launchd.Launchctl{M: t.M, Out: t.Out, Sleep: t.sleep, Label: d.Label,
				Args: []string{"bootout", "system/" + d.Label}})
		}
		for _, path := range []string{d.Plist, d.Dir, d.State, d.Log} {
			var err error
			if cs, err = resource.Append(ctx, cs, t.said(file.Absent{M: t.M, Path: path})); err != nil {
				return nil, err
			}
		}
	}
	return cs, nil
}
