package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

const (
	routesFile   = tunnelEtc + "/routes.d/mac-conf"
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

// Plan plans routes.d/mac-conf.
func (n *Networks) Plan(ctx context.Context) (step.Plan, error) {
	cs, err := n.routes(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	return step.Plan{Changes: cs}, nil
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
	return []step.Change{c}, nil
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

// Apply applies the changes in order.
func (n *Networks) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		if err := Apply(ctx, n.M, n.Out, c); err != nil {
			return err
		}
	}
	return nil
}
