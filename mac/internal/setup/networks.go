package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

const (
	routesFile   = tunnelEtc + "/routes.d/mac-conf"
	resolverDir  = "/etc/resolver"
	routesAbsent = "once routes.d is made"
)

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
	networks := strings.Fields(n.Conf.TunnelNetworks)
	if _, err := n.M.RootAttrs(tunnelEtc + "/routes.d"); errors.Is(err, fs.ErrNotExist) {
		switch {
		case len(networks) == 0:
			return nil, nil
		case n.Check:
			return []step.Change{Pending("would put " + strings.Join(networks, " ") + " in " + routesFile + " " + routesAbsent)}, nil
		}
		return nil, fmt.Errorf("%s/routes.d is missing: run ws's mac/setup.sh first", tunnelEtc)
	} else if err != nil {
		return nil, err
	}
	var c step.Change
	var err error
	if len(networks) > 0 {
		c, err = rootFile(n.M, routesFile, 0o644, []byte(routesText(networks)))
	} else {
		c, err = n.remove(routesFile)
	}
	if err != nil || c == nil {
		return nil, err
	}
	cs := []step.Change{c}
	if _, ok := c.(RootFix); ok || n.Check {
		return cs, nil
	}
	if _, err := n.M.Run(ctx, "launchctl", "print", "system/"+daemonLabel); err == nil {
		cs = append(cs, Launchctl{Args: []string{"kill", "SIGHUP", "system/" + daemonLabel}, Routes: true})
	}
	return cs, nil
}

// remove plans the removal of root's file, if it is there.
func (n *Networks) remove(file string) (step.Change, error) {
	have, err := n.M.ReadRootFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return RootRemove{file, have}, nil
}

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
				"ws tunnel resolver %s with no address, or move the file away", file, r.Domain)
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
	for _, c := range p.Changes {
		var err error
		if l, ok := c.(Launchctl); ok {
			err = launchctl(ctx, n.M, n.Out, time.Sleep, l)
		} else {
			err = Apply(ctx, n.M, n.Out, c)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
