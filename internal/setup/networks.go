package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/step"
)

const resolverDir = "/etc/resolver"

// Networks has the daemon route WS_TUNNEL_NETWORKS, from routes.d/mac-conf, and macOS ask the
// nameservers of WS_RESOLVERS for their domains. A domain whose file lacks ws's header stops the
// run, as does a file that another script wrote. A resolver with the header goes once
// WS_RESOLVERS no longer names its domain.
type Networks struct {
	Conf conf.Conf
	M    effect.Machine
	Out  effect.Out
	// Check is set under --check, which signals no daemon.
	Check bool
}

// Name is networks.
func (*Networks) Name() string { return "networks" }

// WaitsOn is nothing: a check before the tunnel step has run says what would go in routes.d.
func (*Networks) WaitsOn() string { return "" }

// Plan plans, once mac.conf has WS_PUBLIC_KEY and the daemon is to run, routes.d/mac-conf, the signal that has the daemon read it, and /etc/resolver.
func (n *Networks) Plan(ctx context.Context) (step.Plan, error) {
	if n.Conf.PublicKey == "" {
		return step.Plan{}, nil
	}
	cs, err := n.routes(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	rs, err := n.resolvers()
	if err != nil {
		return step.Plan{}, err
	}
	return step.Plan{Changes: append(cs, rs...)}, nil
}

func (n *Networks) routes(ctx context.Context) ([]step.Change, error) {
	r := Routes{Set: "mac-conf", Networks: strings.Fields(n.Conf.TunnelNetworks), M: n.M, Check: n.Check}
	return r.changes(ctx)
}

// remove plans the removal of root's file, if it is there.
func (n *Networks) remove(file string) (step.Change, error) { return removeRoot(n.M, file) }

func (n *Networks) resolvers() ([]step.Change, error) {
	var cs []step.Change
	named := map[string]bool{}
	dirPlanned := false
	for _, r := range resolvers(n.Conf.Resolvers) {
		file := resolverDir + "/" + r.Domain
		named[r.Domain] = true
		if own, err := n.ownsFile(file); err != nil {
			return nil, err
		} else if !own {
			return nil, fmt.Errorf("%s is another script's, or written by hand: have that script call "+
				"ws mac resolver %s with no address, or move the file away", file, r.Domain)
		}
		// The directory is planned once: a second mkdir, with no -p, would fail where it is new.
		var dir step.Change
		if !dirPlanned {
			dirPlanned = true
			var err error
			if dir, err = rootDir(n.M, resolverDir, 0o755); err != nil {
				return nil, err
			}
		}
		file1, err := rootFile(n.M, file, 0o644, []byte(resolverText(r.Addresses)))
		if err != nil {
			return nil, err
		}
		for _, c := range []step.Change{dir, file1} {
			if c != nil {
				cs = append(cs, c)
			}
		}
	}
	names, err := n.M.ReadDir(resolverDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	for _, name := range names {
		file := resolverDir + "/" + name
		if named[name] {
			continue
		}
		if own, err := n.ownsFile(file); err != nil {
			return nil, err
		} else if own {
			c, err := n.remove(file)
			if err != nil {
				return nil, err
			}
			if c != nil {
				cs = append(cs, c)
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
	return applyInOrder(ctx, n.M, n.Out, p)
}
