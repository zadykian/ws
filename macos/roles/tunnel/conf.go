// Package tunnel sets up what the Mac's tunnel daemons read. That is the Mac's WireGuard key, and
// each server's directory and tunnel.conf. launchd's unit of the tunnel runs the daemons.
package tunnel

import (
	"context"
	"path"
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/resource/launchd"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/etcws"
	"github.com/zadykian/ws/macos/facts"
	"github.com/zadykian/ws/macos/values"
)

// Conf is what each server's daemon reads, which ws macos tunnel conf sets up. Each server
// with WS_PUBLIC_KEY gets its directory under /etc/ws/tunnel, routes.d and tunnel.conf. All are
// root's and readable by all. So a run that changes nothing asks for no password, and a root
// daemon reads no file the user can write. A server whose directory in servers/ is gone has
// its daemon taken away. launchd's unit of the tunnel loads and signals the daemons of the rest.
type Conf struct {
	Servers []facts.Server
	// Dir is the directory that holds servers/.
	Dir string
	M   effect.Machine
	Out effect.Out
}

// Resources returns /etc/ws, where a server has WS_PUBLIC_KEY, then each keyed server's files, each directory before what it holds,
// then the prune of the servers that are gone.
func (c Conf) Resources() []resource.Resource {
	var rs []resource.Resource
	for _, s := range c.Servers {
		if s.PublicKey == "" {
			continue
		}
		if rs == nil {
			rs = append(rs, etcws.Top(c.M))
		}
		d := For(s.Name)
		rs = append(rs, etcws.Inside(c.M, d.Conf, []byte(tunnelConf(s.Conf)))...)
		rs = append(rs, file.PublicDir{M: c.M, Path: d.Routes, Mode: etcws.DirMode})
	}
	return append(rs, prune{c})
}

// gone returns the servers whose daemon has a trace on the Mac, and which have no entry in
// servers/. Any entry keeps its daemon, whether it holds a values.yml file or not.
// One moved aside for an edit, or behind a link to a volume not mounted, leaves the server there.
// A listing with no server is not this Mac's, as remove keeps the last one. So a stray WS_VALUES
// removes no live daemon.
func (c Conf) gone() ([]string, error) {
	if len(c.Servers) == 0 {
		return nil, nil
	}
	beside, err := c.M.ReadDir(path.Dir(path.Dir(values.ServerPath(c.Dir, c.Servers[0].Name))))
	if err != nil {
		return nil, err
	}
	found, err := Found(c.M)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(found, func(n string) bool { return !values.IsName(n) || slices.Contains(beside, n) }), nil
}

// prune takes each gone server's daemon away: a bootout where launchd has it loaded, then the
// removal of its directory, state file and log. launchd's unit removes its plist. The daemons left
// read again after, to route what the gone server held: no unit watches a file that is gone.
type prune struct{ c Conf }

// Plan plans the prune, and nothing where no server is gone.
func (p prune) Plan(ctx context.Context) ([]resource.Change, error) {
	gone, err := p.c.gone()
	if err != nil || len(gone) == 0 {
		return nil, err
	}
	var cs []step.Change
	for _, name := range gone {
		d := For(name)
		if loaded(ctx, p.c.M, d.Label) {
			cs = append(cs, bootout{launchd.Launchctl{M: p.c.M}, d.Label})
		}
		for _, path := range []string{d.Dir, d.State, d.Log} {
			if cs, err = resource.Append(ctx, cs, file.Absent{M: p.c.M, Path: path}); err != nil {
				return nil, err
			}
		}
	}
	// launchd's unit takes a gone server's plist away after this one, so a plist alone was pruned
	// and the others read their files again then.
	if len(cs) == 0 {
		return nil, nil
	}
	others, err := Reread(ctx, p.c.M, p.c.Out, gone...)
	if err != nil {
		return nil, err
	}
	cs = append(cs, others...)
	return resource.Planned{Changes: cs}.Plan(ctx)
}

// bootout stops a gone server's daemon and unloads it.
type bootout struct {
	l     launchd.Launchctl
	label string
}

func (b bootout) String() string { return "boot out system/" + b.label }

// Kind is restart.
func (bootout) Kind() step.ChangeKind { return step.Restart }

// Apply boots the daemon out through sudo.
func (b bootout) Apply(ctx context.Context) error { return b.l.Bootout(ctx, b.label) }
