package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// Routes has the daemon route Networks, from the file Set in routes.d: a network a line. With no
// network, it removes the file. The file outlives the daemon, but its routes go with its utun.
type Routes struct {
	Set      string
	Networks []string
	M        effect.Machine
	Out      effect.Out
	// Check is set under --check, which signals no daemon.
	Check bool
}

// Name is tunnel routes.
func (*Routes) Name() string { return "tunnel routes" }

// WaitsOn is nothing.
func (*Routes) WaitsOn() string { return "" }

// Plan plans routes.d/Set.
func (r *Routes) Plan(ctx context.Context) (step.Plan, error) {
	cs, err := r.changes(ctx)
	return step.Plan{Changes: cs}, err
}

// Apply applies the changes in order, so that the file is written before the daemon reads it.
func (r *Routes) Apply(ctx context.Context, p step.Plan) error {
	return applyInOrder(ctx, r.M, r.Out, p)
}

// changes plans routes.d/Set and the signal that has a loaded daemon read it.
func (r *Routes) changes(ctx context.Context) ([]step.Change, error) {
	networks := r.Networks
	file := tunnelDir + "/routes.d/" + r.Set
	if _, err := r.M.RootAttrs(tunnelDir + "/routes.d"); errors.Is(err, fs.ErrNotExist) {
		switch {
		case len(networks) == 0:
			return nil, nil
		case r.Check:
			return []step.Change{Pending("would put " + strings.Join(networks, " ") + " in " + file + " " + routesAbsent)}, nil
		}
		return nil, fmt.Errorf("%s/routes.d is missing: run ws's mac/setup.sh first", tunnelDir)
	} else if err != nil {
		return nil, err
	}
	var c step.Change
	var err error
	if len(networks) > 0 {
		c, err = rootFile(r.M, file, 0o644, []byte(routesText(networks)))
	} else {
		c, err = removeRoot(r.M, file)
	}
	if err != nil || c == nil {
		return nil, err
	}
	cs := []step.Change{c}
	if _, ok := c.(RootFix); ok || r.Check {
		return cs, nil
	}
	if _, err := r.M.Run(ctx, "launchctl", "print", "system/"+daemonLabel); err == nil {
		cs = append(cs, Launchctl{Args: []string{"kill", "SIGHUP", "system/" + daemonLabel}, Routes: true})
	}
	return cs, nil
}
