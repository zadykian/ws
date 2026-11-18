package tunnel

import (
	"context"
	"path"
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	corelaunchd "github.com/zadykian/ws/engine/core/resource/launchd"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/facts"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
	"github.com/zadykian/ws/roles/client/macos/values"
	"github.com/zadykian/ws/roles/internal/etcws"
)

// Conf is what each server's daemon reads, which ws client macos tunnel conf sets up. Each server
// with WS_PUBLIC_KEY gets its directory under /etc/ws/tunnel, routes.d and tunnel.conf. All are
// root's and readable by all. So a run that changes nothing asks for no password, and a root
// daemon reads no file the user can write. A server whose directory beside values.yml is gone has
// its daemon taken away. launchd's unit of the tunnel loads and signals the daemons of the rest.
type Conf struct {
	Servers []facts.Server
	// MacConf is values.yml's path, beside which the servers' directories are.
	MacConf string
	M       effect.Machine
	Out     effect.Out
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
		d := launchd.For(s.Name)
		rs = append(rs, etcws.Inside(c.M, d.Conf, []byte(tunnelConf(s.Conf)))...)
		rs = append(rs, file.PublicDir{M: c.M, Path: d.Routes, Mode: etcws.DirMode})
	}
	return append(rs, prune{c})
}

// gone returns the servers whose daemon has a trace on the Mac, and which have no entry in
// values.yml's directory. Any entry keeps its daemon, whether it holds a values.yml file or not.
// One moved aside for an edit, or behind a link to a volume not mounted, leaves the server there.
// A listing with no server is not this Mac's, as remove keeps the last one. So a stray WS_VALUES
// removes no live daemon.
func (c Conf) gone(ctx context.Context) ([]string, error) {
	if len(c.Servers) == 0 {
		return nil, nil
	}
	beside, err := c.M.ReadDir(path.Dir(path.Dir(values.ServerPath(c.MacConf, c.Servers[0].Name))))
	if err != nil {
		return nil, err
	}
	found, err := launchd.Found(ctx, c.M)
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
	gone, err := p.c.gone(ctx)
	if err != nil || len(gone) == 0 {
		return nil, err
	}
	l := corelaunchd.Launchctl{M: p.c.M}
	var cs []step.Change
	for _, name := range gone {
		d := launchd.For(name)
		if loaded(ctx, p.c.M, d.Label) {
			cs = append(cs, bootout{l, d.Label})
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
	names, err := launchd.Installed(p.c.M)
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		if label := launchd.For(name).Label; !slices.Contains(gone, name) && loaded(ctx, p.c.M, label) {
			cs = append(cs, reread{l, p.c.Out, label})
		}
	}
	return resource.Planned{Changes: cs}.Plan(ctx)
}

// loaded is whether launchd has the daemon loaded, which print shows the user with no root.
func loaded(ctx context.Context, m effect.Machine, label string) bool {
	_, err := m.Run(ctx, "launchctl", "print", "system/"+label)
	return err == nil
}

// bootout stops a gone server's daemon and unloads it.
type bootout struct {
	l     corelaunchd.Launchctl
	label string
}

func (b bootout) String() string { return "boot out system/" + b.label }

// Kind is restart.
func (bootout) Kind() step.ChangeKind { return step.Restart }

// Apply boots the daemon out through sudo.
func (b bootout) Apply(ctx context.Context) error { return b.l.Bootout(ctx, b.label) }

// reread has a loaded daemon read routes.d and the other servers' files again, by SIGHUP. A daemon
// that runs no process reads them as it starts, so a refusal warns.
type reread struct {
	l     corelaunchd.Launchctl
	out   effect.Out
	label string
}

func (r reread) String() string { return "send SIGHUP to system/" + r.label }

// Kind is restart.
func (reread) Kind() step.ChangeKind { return step.Restart }

// Apply signals the daemon through sudo.
func (r reread) Apply(ctx context.Context) error {
	if err := r.l.Signal(ctx, r.label, "SIGHUP"); err != nil {
		r.out.Warn("cannot signal %s, which reads routes.d when it next starts", r.label)
	}
	return nil
}
