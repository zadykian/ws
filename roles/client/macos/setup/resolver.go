package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/check"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
)

// resolverCmdHeader is the first line of the files that ws client macos resolver writes. Those of
// the networks step have their own, so each owner leaves the other's.
const resolverCmdHeader = "# Written by github.com/zadykian/ws: ws client macos resolver, which removes this file when given no address."

// Resolver has macOS ask the nameservers Addresses, up to 3, for Domain and the names under it, and
// the Mac's own for the rest. It writes /etc/resolver/Domain (resolver(5)). With no address, it
// removes the file, but only one that has the header. It stops rather than replace such a file of
// another's.
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
func (r *Resolver) Plan(ctx context.Context) (step.Plan, error) {
	if !check.Name(r.Domain) {
		return step.Plan{}, fmt.Errorf("'%s' is not a domain of DNS labels joined by dots", r.Domain)
	}
	for _, a := range r.Addresses {
		if !check.IPv4(a) {
			return step.Plan{}, fmt.Errorf("%s: '%s' is not an IPv4 address", r.Domain, a)
		}
	}
	if len(r.Addresses) > 3 {
		return step.Plan{}, fmt.Errorf("%s: macOS takes 3 nameservers for a domain, not %d",
			r.Domain, len(r.Addresses))
	}
	file := resolverDir + "/" + r.Domain
	text, err := r.M.ReadRootFile(file)
	exists := !errors.Is(err, fs.ErrNotExist)
	if err != nil && exists {
		return step.Plan{}, err
	}
	if first, _, _ := strings.Cut(string(text), "\n"); exists && first != resolverCmdHeader {
		if len(r.Addresses) > 0 {
			return step.Plan{}, fmt.Errorf("%s is there, and ws client macos resolver didn't write it: move it away first", file)
		}
		r.Out.Say("leaves %s, which ws client macos resolver didn't write", file)
		return step.Plan{}, nil
	}
	if len(r.Addresses) == 0 {
		cs, err := change.Plan(ctx, nil, host.Absent{M: r.M, Path: file})
		return step.Plan{Changes: cs}, err
	}
	cs, err := change.Plan(ctx, nil, host.Dir{M: r.M, Path: resolverDir, Mode: 0o755})
	if err != nil {
		return step.Plan{}, err
	}
	want := resolverCmdHeader + "\nnameserver " + strings.Join(r.Addresses, "\nnameserver ") + "\n"
	cs, err = change.Plan(ctx, cs, host.File{M: r.M, Path: file, Content: []byte(want), Mode: 0o644})
	return step.Plan{Changes: cs}, err
}

// Apply applies the changes in order, so that the directory is made before the file.
func (r *Resolver) Apply(ctx context.Context, p step.Plan) error {
	return applyInOrder(ctx, r.M, r.Out, p)
}
