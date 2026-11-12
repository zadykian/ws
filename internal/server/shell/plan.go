package shell

import (
	"context"

	"github.com/zadykian/ws/host"
)

// planAll joins the changes of resources, in order.
func planAll(ctx context.Context, rs ...host.Resource) ([]host.Change, error) {
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
