package cli

import (
	"fmt"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/setup"
)

// UpdateSetups is the setup that ws update ends in on Ubuntu, the guest's or the host's. The side's
// env in the miserc, which a setup wrote, picks it. A machine with no miserc has run none, so the
// update runs no setup there.
func UpdateSetups() enginecli.Setups {
	return enginecli.Setups{
		Names: []string{"ws ubuntu guest setup", "ws ubuntu host setup"},
		Pick: func(m effect.Machine) ([]string, error) {
			envs, err := setup.SideEnvs(m)
			if err != nil || len(envs) == 0 {
				return nil, err
			}
			if len(envs) != 1 || (envs[0] != "guest" && envs[0] != "host") {
				return nil, fmt.Errorf("%s names the envs %v, and ws update knows guest or host alone", setup.MisercPath, envs)
			}
			return []string{"ubuntu", envs[0], "setup"}, nil
		},
	}
}
