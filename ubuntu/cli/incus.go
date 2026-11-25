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
		incusIngress(p),
		incusObject(p, "pool", "make Incus's ZFS pool on the partition of WS_INCUS_POOL_SOURCE",
			"It reads the pool from incus, and creates it where it is missing, from the keys that ws declares. "+
				"It fails where the pool was made on another partition, as Incus never moves a pool.",
			func(v values.Incus) resource.Resource { return incus.Pool(p.Machine, v) }),
		incusObject(p, "bridge", "make the guests' bridge, with the ACLs on it",
			"It creates incusbr0 where it is missing, and edits the keys that ws declares where they differ: its "+
				"subnet, its DHCP's resolvers, NAT, no IPv6 and the ACLs with egress and ingress dropped by default.",
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

// incusIngress returns ws ubuntu host incus ingress: the ACL of what reaches the guests. It reads each
// guest's own values for its tunnel, so it fails where two tunnels clash.
func incusIngress(p enginecli.Env) *cobra.Command {
	return procedure("ingress", "make the ingress ACL of the guests",
		"It reads the ACL from incus, and creates it where it is missing, or edits its ingress rules where they "+
			"differ. It allows each peer of each guest's tunnel to reach the guest's own addresses, and a peer of "+
			"WS_IKEWARD_FORWARD_FROM to reach any destination. The bridge drops the rest.",
		func(context.Context, effect.Out) ([]resource.Resource, error) {
			t, err := guestTunnels(p)
			if err != nil {
				return nil, err
			}
			return []resource.Resource{incus.IngressACL(p.Machine, t)}, nil
		})
}

// incusGuests returns ws ubuntu host incus guests. It launches or edits each guest of the host's
// list, starts a stopped one, and sets each up.
func incusGuests(p enginecli.Env) *cobra.Command {
	return procedure("guests", "launch, edit, start and set up each guest of WS_INCUS_GUESTS",
		"It reads each guest from incus, launches the one that is missing from the image, and edits the keys "+
			"that ws declares where they differ: its limits, its NIC at its address and its disk's quota. It starts "+
			"a guest that is stopped. A guest of ws's that the list lacks gets a warning, and is never deleted.\n\n"+
			"It then pushes the host's ws and the guest's own values into each guest where they differ, and runs "+
			"the guest's ws ubuntu guest setup where the guest's own plan finds a change, for up to an hour.",
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
