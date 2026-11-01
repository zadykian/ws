package wizard

import (
	"context"
	"errors"

	"github.com/zadykian/ws/mac/internal/step"
)

// Plan finds what gh lacks: its install, a login with the scopes, and the Mac's key for ws among
// the login's keys. It reads no keys once it would log in.
func (g *Gh) Plan(ctx context.Context) (step.Plan, error) {
	path, cs, err := g.ghPath(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	login := len(cs) > 0
	if !login {
		_, err := g.M.Run(ctx, path, "auth", "status", "-h", "github.com")
		login = err != nil
	}
	pub, key, err := g.key(ctx)
	var missing noKey
	if errors.As(err, &missing) {
		// The Mac's key is yet to be made: say so under --check, and fail in Apply, as sh did.
		if login {
			cs = append(cs, ghLogin{[]string{"login", "-p", "https", "-w"}})
		}
		return step.Plan{Changes: append(cs, ghLater{missing})}, nil
	}
	if err != nil {
		return step.Plan{}, err
	}
	add := ghAdd{pub: pub, title: g.hostname(ctx)}
	if login {
		return step.Plan{Changes: append(cs, ghLogin{[]string{"login", "-p", "https", "-w"}}, add)}, nil
	}
	listed, refresh, err := g.listed(ctx, path, key)
	switch {
	case err != nil:
		return step.Plan{}, err
	case refresh:
		return step.Plan{Changes: []step.Change{ghLogin{[]string{"refresh"}}, add}}, nil
	case !listed:
		return step.Plan{Changes: []step.Change{add}}, nil
	}
	return step.Plan{}, nil
}
