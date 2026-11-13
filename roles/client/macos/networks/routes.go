package networks

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/check"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
	tunnel "github.com/zadykian/ws/roles/client/macos/tunnel/plan"
)

const routesAbsent = "once routes.d is made"

// Routes has the daemon route Networks, from the file Set in routes.d: a network a line. With no
// network, it removes the file. The file outlives the daemon, but its routes go with its utun.
type Routes struct {
	Set      string
	Networks []string
	M        effect.Machine
	Out      effect.Out
	// Check is set under --check, which signals no daemon.
	Check bool
	// Alone is set where no step makes routes.d first: a missing routes.d is an error even under
	// --check, as the old script's was.
	Alone bool
}

// Name is tunnel routes.
func (*Routes) Name() string { return "tunnel routes" }

// WaitsOn is nothing.
func (*Routes) WaitsOn() string { return "" }

// Plan plans routes.d/Set.
func (r *Routes) Plan(ctx context.Context) (step.Plan, error) {
	if err := r.validate(); err != nil {
		return step.Plan{}, err
	}
	cs, err := r.changes(ctx)
	return step.Plan{Changes: cs}, err
}

// Apply applies the changes in order, so that the file is written before the daemon reads it.
func (r *Routes) Apply(ctx context.Context, p step.Plan) error {
	return resource.Group{}.Apply(ctx, p)
}

// changes plans routes.d/Set, and the signal that has a loaded daemon read it. A change of the
// file's owner or mode alone sends the signal too.
func (r *Routes) changes(ctx context.Context) ([]step.Change, error) {
	networks := r.Networks
	routesFile := launchd.Dir + "/routes.d/" + r.Set
	if _, err := r.M.RootAttrs(launchd.Dir + "/routes.d"); errors.Is(err, fs.ErrNotExist) {
		switch {
		case len(networks) == 0:
			return nil, nil
		case r.Check && !r.Alone:
			return []step.Change{resource.Remark("would put " + strings.Join(networks, " ") + " in " + routesFile + " " + routesAbsent)}, nil
		}
		return nil, fmt.Errorf("%s/routes.d is missing: run ws client macos apply first", launchd.Dir)
	} else if err != nil {
		return nil, err
	}
	var res resource.Resource = file.Absent{M: r.M, Path: routesFile}
	if len(networks) > 0 {
		res = file.File{M: r.M, Path: routesFile, Content: []byte(routesText(networks)), Mode: 0o644}
	}
	cs, err := resource.Append(ctx, nil, resource.Said{Out: r.Out, Part: res})
	if err != nil || len(cs) == 0 || r.Check {
		return cs, err
	}
	if _, err := r.M.Run(ctx, "launchctl", "print", "system/"+launchd.Label); err == nil {
		cs = append(cs, launchd.Launchctl{M: r.M, Out: r.Out,
			Args: []string{"kill", "SIGHUP", "system/" + launchd.Label}, Routes: true})
	}
	return cs, nil
}

// validate refuses a set that is not of a-z, 0-9 and -. It refuses a network that is not an IPv4
// network, or that overlaps the tunnel's or the cable's, which 0.0.0.0/0 does too.
func (r *Routes) validate() error {
	if r.Set == "" || strings.Trim(r.Set, "abcdefghijklmnopqrstuvwxyz0123456789-") != "" {
		return fmt.Errorf("'%s' is not a name of a-z, 0-9 and -", r.Set)
	}
	for _, n := range r.Networks {
		p, ok := check.CIDR(n)
		if !ok {
			return fmt.Errorf("%s: '%s' is not an IPv4 network, as 192.0.2.0/24", r.Set, n)
		}
		if p.Overlaps(tunnel.TunnelNet) {
			return fmt.Errorf("%s: %s overlaps the tunnel's %s", r.Set, n, tunnel.TunnelNet)
		}
		if p.Overlaps(tunnel.CableNet) {
			return fmt.Errorf("%s: %s overlaps the cable's %s", r.Set, n, tunnel.CableNet)
		}
	}
	return nil
}
