package cli

import (
	"fmt"
	"slices"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/setup"
)

// UpdateSetups is the setup that ws update ends in on Ubuntu, the guest's or the host's. The side's
// env in the miserc, which a setup wrote, picks it. The mode's env, which may stand beside it, names no side. A machine
// with no miserc has run none, so the update runs no setup there.
func UpdateSetups() enginecli.Setups {
	return enginecli.Setups{
		Names: []string{"ws ubuntu guest setup", "ws ubuntu host setup"},
		Pick: func(m effect.Machine) ([]string, error) {
			envs, err := setup.SideEnvs(m)
			if err != nil || len(envs) == 0 {
				return nil, err
			}
			sides := slices.DeleteFunc(slices.Clone(envs), func(e string) bool { return e == setup.SelfContained })
			if len(sides) != 1 || len(envs)-len(sides) > 1 || (sides[0] != "guest" && sides[0] != "host") {
				return nil, fmt.Errorf("%s names the envs %v, and ws update knows guest or host, with %s, alone",
					setup.MisercPath, envs, setup.SelfContained)
			}
			return []string{"ubuntu", sides[0], "setup"}, nil
		},
	}
}
