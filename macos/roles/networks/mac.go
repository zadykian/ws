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
	"github.com/zadykian/ws/macos/facts"
)

// Mac is the networks of each of the Mac's servers that has WS_PUBLIC_KEY, which ws macos
// networks sets up. Each server's daemon routes its WS_TUNNEL_NETWORKS, from its routes.d/values,
// and macOS asks the nameservers of its WS_RESOLVERS for their domains. A domain whose file lacks
// ws's header stops the run, as does a file that another script wrote. A resolver with the header
// goes once no server's file names its domain. launchd's unit of the tunnel watches routes.d, so
// a change here signals no daemon. The command says each change as it applies.
type Mac struct {
	Servers []facts.Server
	M       effect.Machine
	Out     effect.Out
}

// Plan plans each server's routes.d/values, then /etc/resolver.
func (n Mac) Plan(ctx context.Context) ([]resource.Change, error) {
	var cs []step.Change
	for _, s := range n.Servers {
		if s.PublicKey == "" {
			continue
		}
		r := Routes{Server: s.Name, Set: "values", Networks: strings.Fields(s.TunnelNetworks), M: n.M, Out: n.Out, Watched: true}
		more, err := r.changes(ctx)
		if err != nil {
			return nil, err
		}
		cs = append(cs, more...)
	}
	rs, err := n.resolvers(ctx)
	if err != nil {
		return nil, err
	}
	return resource.Planned{Changes: append(cs, rs...)}.Plan(ctx)
}

// resolvers plans the files of /etc/resolver: one a domain of each server with WS_PUBLIC_KEY, and
// the removal of each of ws's that no server's file names.
func (n Mac) resolvers(ctx context.Context) ([]step.Change, error) {
	var cs []step.Change
	named := map[string]bool{}
	for _, s := range n.Servers {
		for _, r := range resolvers(s.Resolvers) {
			named[r.Domain] = true
			if s.PublicKey == "" {
				continue
			}
			resolverFile := resolverDir + "/" + r.Domain
			if own, err := ownsFile(n.M, resolverFile); err != nil {
				return nil, err
			} else if !own {
				return nil, fmt.Errorf("%s is another script's, or written by hand: have that script call "+
					"ws macos resolver %s with no address, or move the file away", resolverFile, r.Domain)
			}
			var err error
			// The directory is planned once: a second mkdir would fail where the directory is new.
			if len(cs) == 0 {
				if cs, err = resource.Append(ctx, cs, file.Dir{M: n.M, Path: resolverDir, Mode: 0o755}); err != nil {
					return nil, err
				}
			}
			if cs, err = resource.Append(ctx, cs, file.File{M: n.M, Path: resolverFile,
				Content: []byte(resolverText(r.Addresses)), Mode: 0o644}); err != nil {
				return nil, err
			}
		}
	}
	names, err := n.M.ReadDir(resolverDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	for _, name := range names {
		resolverFile := resolverDir + "/" + name
		if named[name] {
			continue
		}
		if own, err := ownsFile(n.M, resolverFile); err != nil {
			return nil, err
		} else if !own {
			continue
		}
		gone, err := file.Absent{M: n.M, Path: resolverFile}.Plan(ctx)
		if err != nil {
			return nil, err
		}
		for _, c := range gone {
			cs = append(cs, removal{c})
		}
	}
	return cs, nil
}
