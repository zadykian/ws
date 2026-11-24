package cli

import (
	"context"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/ubuntu/roles/incus"
	"github.com/zadykian/ws/ubuntu/values"
)

// newIncusCmd returns ws ubuntu host incus: the commands that the incus role's units run, one for
// each object that ws makes in Incus.
func newIncusCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Group("incus", "the incus role's objects, which its units make", incusObject(p, "pool",
		"make Incus's ZFS pool on the partition of WS_INCUS_POOL_SOURCE",
		"It reads the pool from incus, and creates it where it is missing, from the keys that ws declares. "+
			"It fails where the pool was made on another partition, as Incus never moves a pool.",
		func(v values.Incus) resource.Resource { return incus.Pool(p.Machine, v) }))
}

// incusObject returns a command of incus's that settles the object that obj gives for the host's
// values, with --check.
func incusObject(p enginecli.Env, use, short, long string, obj func(values.Incus) resource.Resource) *cobra.Command {
	return procedure(use, short, long, func(_ context.Context, _ effect.Out) ([]resource.Resource, error) {
		v, err := values.LoadHost(p.Machine, hostVars)
		if err != nil {
			return nil, err
		}
		return []resource.Resource{obj(v.Incus)}, nil
	})
}
