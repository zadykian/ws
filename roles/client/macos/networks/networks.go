// Package networks sets up the routes and resolvers of the networks that the tunnel reaches.
package networks

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
)

const resolverDir = "/etc/resolver"

// Networks has the daemon of Server route WS_TUNNEL_NETWORKS, from its routes.d/server-conf, and
// macOS ask the nameservers of WS_RESOLVERS for their domains. A domain whose file lacks ws's
// header stops the run, as does a file that another script wrote. A resolver with the header goes
// once WS_RESOLVERS no longer names its domain, unless another server's file names it.
type Networks struct {
	Server string
	// MacConf is mac.conf's path, whose servers/ holds the other servers' files.
	MacConf string
	Conf    conf.Conf
	M       effect.Machine
	Out     effect.Out
	// Check is set under --check, which signals no daemon.
	Check bool
}

// Name is networks.
func (*Networks) Name() string { return "networks" }

// WaitsOn is nothing: a check before the tunnel step has run says what would go in routes.d.
func (*Networks) WaitsOn() string { return "" }

// Plan plans, once mac.conf has WS_PUBLIC_KEY and the daemon is to run, routes.d/server-conf, the
// signal that has the daemon read it, and /etc/resolver.
func (n *Networks) Plan(ctx context.Context) (step.Plan, error) {
	if n.Conf.PublicKey == "" {
		return step.Plan{}, nil
	}
	cs, err := n.routes(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	rs, err := n.resolvers(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	return step.Plan{Changes: append(cs, rs...)}, nil
}

// Report says how routes.d/server-conf and the resolvers that the server's file names stand. A file a run is to
// remove comes from its change, as mac.conf no longer names it.
func (n *Networks) Report(ctx context.Context) ([]step.Item, error) {
	if n.Conf.PublicKey == "" {
		return nil, nil
	}
	var parts []step.Reporter
	if nets := strings.Fields(n.Conf.TunnelNetworks); len(nets) > 0 {
		parts = append(parts, file.File{M: n.M, Path: launchd.For(n.Server).Routes + "/server-conf", Content: []byte(routesText(n.Server, "server-conf", nets)), Mode: 0o644})
	}
	list := resolvers(n.Conf.Resolvers)
	if len(list) > 0 {
		parts = append(parts, file.Dir{M: n.M, Path: resolverDir, Mode: 0o755})
	}
	for _, r := range list {
		parts = append(parts, file.File{M: n.M, Path: resolverDir + "/" + r.Domain, Content: []byte(resolverText(r.Addresses)), Mode: 0o644})
	}
	return file.Gather(ctx, parts...)
}

func (n *Networks) routes(ctx context.Context) ([]step.Change, error) {
	r := Routes{Server: n.Server, Set: "server-conf", Networks: strings.Fields(n.Conf.TunnelNetworks), M: n.M, Out: n.Out, Check: n.Check}
	return r.changes(ctx)
}

func (n *Networks) resolvers(ctx context.Context) ([]step.Change, error) {
	var cs []step.Change
	var err error
	named := map[string]bool{}
	for i, r := range resolvers(n.Conf.Resolvers) {
		resolverFile := resolverDir + "/" + r.Domain
		named[r.Domain] = true
		if own, err := n.ownsFile(resolverFile); err != nil {
			return nil, err
		} else if !own {
			return nil, fmt.Errorf("%s is another script's, or written by hand: have that script call "+
				"ws client macos resolver %s with no address, or move the file away", resolverFile, r.Domain)
		}
		// The directory is planned once: a second mkdir would fail where the directory is new.
		if i == 0 {
			if cs, err = resource.Append(ctx, cs, resource.Said{Out: n.Out, Part: file.Dir{M: n.M, Path: resolverDir, Mode: 0o755}}); err != nil {
				return nil, err
			}
		}
		if cs, err = resource.Append(ctx, cs, resource.Said{Out: n.Out,
			Part: file.File{M: n.M, Path: resolverFile, Content: []byte(resolverText(r.Addresses)), Mode: 0o644}}); err != nil {
			return nil, err
		}
	}
	names, err := n.M.ReadDir(resolverDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	siblings, err := conf.Siblings(n.M, n.MacConf, conf.ServerPath(n.MacConf, n.Server))
	if err != nil {
		return nil, err
	}
	for _, s := range siblings {
		for _, r := range resolvers(s.Conf.Resolvers) {
			named[r.Domain] = true
		}
	}
	for _, name := range names {
		resolverFile := resolverDir + "/" + name
		if named[name] {
			continue
		}
		if own, err := n.ownsFile(resolverFile); err != nil {
			return nil, err
		} else if own {
			gone, err := resource.Said{Out: n.Out, Part: file.Absent{M: n.M, Path: resolverFile}}.Plan(ctx)
			if err != nil {
				return nil, err
			}
			for _, c := range gone {
				cs = append(cs, removal{c, resolverFile})
			}
		}
	}
	return cs, nil
}

// ownsFile is whether the file is missing or starts with ws's header.
func (n *Networks) ownsFile(file string) (bool, error) {
	have, err := n.M.ReadRootFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	first, _, _ := strings.Cut(string(have), "\n")
	return first == resolverHeader, nil
}

// Apply applies the changes in order, so that routes.d is written before the daemon reads it.
func (n *Networks) Apply(ctx context.Context, p step.Plan) error {
	return resource.Group{}.Apply(ctx, p)
}

// removal is host's removal of a file that no Report names, as mac.conf no longer does. It says
// what the file's Report would.
type removal struct {
	resource.Change
	path string
}

// Kind is removal.
func (removal) Kind() step.ChangeKind { return step.Removal }

// Item says that the file is to go.
func (r removal) Item() step.Item {
	return step.Item{Kind: step.FileItem, Name: r.path, Want: file.StateMissing, Found: file.StatePresent}
}
