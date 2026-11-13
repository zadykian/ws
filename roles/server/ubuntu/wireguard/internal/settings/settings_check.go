package settings

import (
	"context"

	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// Check refuses the values the role cannot use, as a plan error.
type Check struct{ W values.Wireguard }

// Plan fails with the refusal of the settings check, or plans nothing.
func (c Check) Plan(context.Context) ([]host.Change, error) { return nil, Validate(c.W) }
