package wireguard

import (
	"context"

	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// settingsCheck refuses the values the role cannot use, as a plan error.
type settingsCheck struct{ w values.Wireguard }

func (c settingsCheck) Plan(context.Context) ([]host.Change, error) { return nil, Settings(c.w) }
