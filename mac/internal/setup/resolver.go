package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/mac/internal/check"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// resolverCmdHeader is the first line of the files that ws resolver writes. Those of the networks
// step have their own, so each owner leaves the other's.
const resolverCmdHeader = "# Written by ws resolver, which removes this file when given no address."

// Resolver has macOS ask the nameservers Addresses, up to 3, for Domain and the names under it, and
// the Mac's own for the rest: it writes /etc/resolver/Domain (resolver(5)).
type Resolver struct {
	Domain    string
	Addresses []string
	M         effect.Machine
	Out       effect.Out
}

// Name is resolver.
func (*Resolver) Name() string { return "resolver" }

// WaitsOn is nothing.
func (*Resolver) WaitsOn() string { return "" }

// Plan plans /etc/resolver/Domain.
func (r *Resolver) Plan(context.Context) (step.Plan, error) {
	if !check.Name(r.Domain) {
		return step.Plan{}, fmt.Errorf("ws resolver: '%s' is not a domain of DNS labels joined by dots", r.Domain)
	}
	for _, a := range r.Addresses {
		if !check.IPv4(a) {
			return step.Plan{}, fmt.Errorf("ws resolver %s: '%s' is not an IPv4 address", r.Domain, a)
		}
	}
	if len(r.Addresses) > 3 {
		return step.Plan{}, fmt.Errorf("ws resolver %s: macOS takes 3 nameservers for a domain, not %d",
			r.Domain, len(r.Addresses))
	}
	file := resolverDir + "/" + r.Domain
	if len(r.Addresses) == 0 {
		return step.Plan{}, nil
	}
	dir, err := rootDir(r.M, resolverDir, 0o755)
	if err != nil {
		return step.Plan{}, err
	}
	want := resolverCmdHeader + "\nnameserver " + strings.Join(r.Addresses, "\nnameserver ") + "\n"
	c, err := rootFile(r.M, file, 0o644, []byte(want))
	var cs []step.Change
	for _, c := range []step.Change{dir, c} {
		if c != nil {
			cs = append(cs, c)
		}
	}
	return step.Plan{Changes: cs}, err
}

// Apply applies the changes in order, so that the directory is made before the file.
func (r *Resolver) Apply(ctx context.Context, p step.Plan) error {
	return applyInOrder(ctx, r.M, r.Out, p)
}
