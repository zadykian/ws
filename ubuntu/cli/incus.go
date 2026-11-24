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
	return enginecli.Group("incus", "the incus role's objects, which its units make",
		incusObject(p, "acl", "make the egress ACL of the guests",
			"It reads the ACL from incus, and creates it where it is missing, or edits its description and its egress "+
				"rules where they differ. It drops the private ranges and allows the rest.",
			func(values.Incus) resource.Resource { return incus.EgressACL(p.Machine) }),
		incusObject(p, "pool", "make Incus's ZFS pool on the partition of WS_INCUS_POOL_SOURCE",
			"It reads the pool from incus, and creates it where it is missing, from the keys that ws declares. "+
				"It fails where the pool was made on another partition, as Incus never moves a pool.",
			func(v values.Incus) resource.Resource { return incus.Pool(p.Machine, v) }),
		incusObject(p, "bridge", "make the guests' bridge, with the egress ACL on it",
			"It creates incusbr0 where it is missing, and edits the keys that ws declares where they differ: its "+
				"subnet, its DHCP's resolvers, NAT, no IPv6 and the ACL with egress dropped by default.",
			func(v values.Incus) resource.Resource { return incus.Bridge(p.Machine, v) }),
		incusObject(p, "profile", "make the profile that each guest takes",
			"It creates the profile ws where it is missing, and edits the keys and devices that ws declares where "+
				"they differ: unprivileged, nesting, an isolated NIC on the bridge and the disk on the pool.",
			func(values.Incus) resource.Resource { return incus.Profile(p.Machine) }),
		incusObject(p, "image", "import the image that the guests launch from",
			"It downloads Ubuntu's root tarball of the pinned release, checked against its SHA-256, writes the "+
				"metadata with the cloud-init seed's templates, and imports both under an alias that names the release "+
				"and the metadata. Where Incus holds the alias, it does nothing.",
			func(values.Incus) resource.Resource { return incus.Image{M: p.Machine} }),
		incusObject(p, "owned", "fill the table of sources with each guest's address",
			"It reads the set owned4 of the table ws_sources, and adds the address and port of each guest of "+
				"WS_INCUS_GUESTS that it lacks, and deletes an element that no guest holds. A reload of the table "+
				"empties the set, so each run fills it again.",
			func(v values.Incus) resource.Resource { return incus.Owned{M: p.Machine, Guests: v.Guests} }),
		incusGuests(p))
}

// incusGuests returns ws ubuntu host incus guests. It launches or edits each guest of the host's
// list, starts a stopped one, and sets each up.
func incusGuests(p enginecli.Env) *cobra.Command {
	return procedure("guests", "launch, edit and start each guest of WS_INCUS_GUESTS",
		"It reads each guest from incus, launches the one that is missing from the image, and edits the keys "+
			"that ws declares where they differ: its limits, its NIC at its address and its disk's quota. It starts "+
			"a guest that is stopped. A guest of ws's that the list lacks gets a warning, and is never deleted.",
		func(ctx context.Context, out effect.Out) ([]resource.Resource, error) {
			v, err := values.LoadHost(p.Machine, hostVars)
			if err != nil {
				return nil, err
			}
			return incus.Guests(ctx, p.Machine, v.Incus.Guests, out)
		})
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
