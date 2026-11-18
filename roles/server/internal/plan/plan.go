// Package plan joins the changes of resources, as git's tokens plan a directory and its files.
package plan

import (
	"context"

	"github.com/zadykian/ws/engine/core/resource"
)

// All joins the changes of resources, in order.
func All(ctx context.Context, rs ...resource.Resource) ([]resource.Change, error) {
	var all []resource.Change
	for _, r := range rs {
		changes, err := r.Plan(ctx)
		if err != nil {
			return nil, err
		}
		all = append(all, changes...)
	}
	return all, nil
}
