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
	"github.com/zadykian/ws/internal/cablenet"
	"github.com/zadykian/ws/internal/slot"
	"github.com/zadykian/ws/macos/roles/tunnel"
)

const routesAbsent = "once routes.d is made"

// Routes has the daemon of Server route Networks, from the file Set in its routes.d: a network a
// line. With no network, it removes the file. The file outlives the daemon, but its routes go with
// its utun.
type Routes struct {
	Server   string
	Set      string
	Networks []string
	M        effect.Machine
	Out      effect.Out
	// Check is set under --check, which signals no daemon.
	Check bool
	// Alone is set where no step makes routes.d first: a missing routes.d is an error even under
	// --check, as the old script's was.
	Alone bool
	// Watched is set where launchd's unit of the tunnel watches routes.d, and so signals the
	// daemons itself. A missing routes.d is then due, as the tunnel's unit makes it first.
	Watched bool
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

// changes plans routes.d/Set, and the signal that has each loaded daemon read it. A change of the
// file's owner or mode alone sends the signal too.
func (r *Routes) changes(ctx context.Context) ([]step.Change, error) {
	networks := r.Networks
	dir := tunnel.For(r.Server).Routes
	routesFile := dir + "/" + r.Set
	if _, err := r.M.RootAttrs(dir); errors.Is(err, fs.ErrNotExist) {
		switch {
		case len(networks) == 0:
			return nil, nil
		case (r.Check || r.Watched) && !r.Alone:
			return []step.Change{resource.Remark("would put " + strings.Join(networks, " ") + " in " + routesFile + " " + routesAbsent)}, nil
		}
		return nil, fmt.Errorf("%s is missing: run mise -C /opt/ws/macos bootstrap first", dir)
	} else if err != nil {
		return nil, err
	}
	var res resource.Resource = file.Absent{M: r.M, Path: routesFile}
	if len(networks) > 0 {
		res = file.File{M: r.M, Path: routesFile, Content: []byte(routesText(r.Server, r.Set, networks)), Mode: 0o644}
	}
	if !r.Watched {
		// A command of the Mac's says each change as it applies, so a watched file needs no Said.
		res = resource.Said{Out: r.Out, Part: res}
	}
	cs, err := resource.Append(ctx, nil, res)
	if err != nil || len(cs) == 0 || r.Check || r.Watched {
		return cs, err
	}
	reread, err := tunnel.Reread(ctx, r.M, r.Out)
	if err != nil {
		return nil, err
	}
	cs = append(cs, reread...)
	return cs, nil
}

// validate refuses a set that is not of a-z, 0-9 and -. It refuses a network that is not an IPv4
// network, or that overlaps the tunnels' or the cable's, which 0.0.0.0/0 does too.
func (r *Routes) validate() error {
	if r.Set == "" || strings.Trim(r.Set, "abcdefghijklmnopqrstuvwxyz0123456789-") != "" {
		return fmt.Errorf("'%s' is not a name of a-z, 0-9 and -", r.Set)
	}
	for _, n := range r.Networks {
		p, ok := check.CIDR(n)
		if !ok {
			return fmt.Errorf("%s: '%s' is not an IPv4 network, as 192.0.2.0/24", r.Set, n)
		}
		if p.Overlaps(slot.Range) {
			return fmt.Errorf("%s: %s overlaps the tunnels' %s", r.Set, n, slot.Range)
		}
		if p.Overlaps(cablenet.Net) {
			return fmt.Errorf("%s: %s overlaps the cable's %s", r.Set, n, cablenet.Net)
		}
	}
	return nil
}
