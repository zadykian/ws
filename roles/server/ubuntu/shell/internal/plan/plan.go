// Package plan holds what the shell role's parts share: joining the changes of resources and
// naming a path below the account's home.
package plan

import (
	"context"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/host"
)

// All joins the changes of resources, in order.
func All(ctx context.Context, rs ...host.Resource) ([]host.Change, error) {
	var all []host.Change
	for _, r := range rs {
		changes, err := r.Plan(ctx)
		if err != nil {
			return nil, err
		}
		all = append(all, changes...)
	}
	return all, nil
}

// Below returns p as a path below the account's home, both cleaned as values cleans them, and empty
// for the home itself. A path outside the home is returned whole, and the home file refuses it.
func Below(a host.Account, p string) string {
	home, p := path.Clean(a.Home), path.Clean(p)
	if p == home {
		return ""
	}
	return strings.TrimPrefix(p, strings.TrimSuffix(home, "/")+"/")
}
