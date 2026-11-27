package incus

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/ubuntu/roles/incus/object"
	"github.com/zadykian/ws/ubuntu/values"
)

// guestKind is incus's instance. launch makes one from the image and starts it, and config edit
// replaces the writable keys, which are the instance's own, less what Incus fills.
func guestKind(alias string) object.Kind {
	return object.Kind{
		Noun:   "guest",
		List:   []string{"list"},
		Create: []string{"launch", "local:" + alias, object.NameArg},
		Edit:   []string{"config", "edit", object.NameArg},
		Keys:   []string{"description", "architecture", "config", "devices", "ephemeral", "profiles", "stateful"},
	}
}

// Guest is the instance of the guest that tunnel t names. It holds the profile ws, its limits, its
// NIC with its slot's address as a route, and its disk. Where gpu is on, it holds the host's NVIDIA
// card too. Each device is whole, as a device of an instance replaces the profile's. The seed is
// a key of the create alone. An instance with no profile ws is no guest of ws's.
func Guest(m effect.Machine, t values.GuestTunnel, gpu bool, alias string) object.Object {
	g := t.Guest
	return object.Object{
		M: m, Kind: guestKind(alias), Name: g.Name,
		Want: map[string]any{
			"description": "ws's guest " + g.Name,
			"profiles":    []any{ProfileName},
			"config":      map[string]any{"limits.cpu": strconv.Itoa(g.CPU), "limits.memory": g.Memory},
			"devices": map[string]any{
				"eth0": nicDevice(nicKeys(t)),
				"root": rootDevice(g.Disk),
				"gpu":  gpuDevice(gpu),
			},
		},
		First: map[string]any{"config": map[string]any{"cloud-init.network-config": seedNetwork(t)}},
		Check: func(have map[string]any) error {
			if profiles, _ := have["profiles"].([]any); !slices.Contains(profiles, any(ProfileName)) {
				return fmt.Errorf("it takes no profile %s: it is no guest of ws's, and ws never turns an instance into one", ProfileName)
			}
			return nil
		},
	}
}

// nvidiaVendor is NVIDIA's PCI vendor ID, as Incus's gpu device names it.
const nvidiaVendor = "10de"

// gpuDevice gives a guest NVIDIA's card, which several guests may hold at once, time-sliced and
// with no memory isolation between them. Incus passes /dev/nvidia* as they exist when the guest
// starts, and the role nvidia waits for them. With the switch off the device is nil, which removes it.
func gpuDevice(on bool) any {
	if !on {
		return nil
	}
	return map[string]any{"type": "gpu", "vendorid": nvidiaVendor}
}

// nicKeys are the NIC's own keys: its port's name and its address, and the slot's address as a
// route. Incus then sends that address to this NIC, which is where the host's tunnel for the guest
// delivers.
func nicKeys(t values.GuestTunnel) map[string]any {
	return map[string]any{"host_name": t.Guest.Port(), "ipv4.address": t.Guest.Address, "ipv4.routes": t.SlotAddress().String() + "/32"}
}

// seedNetwork is the guest's network-config. It holds DHCP on eth0, which gives the NIC's address,
// and the slot's address. The tunnel ends on the host, which routes that address to the guest. So
// the guest holds it as a host route.
func seedNetwork(t values.GuestTunnel) string {
	return fmt.Sprintf("version: 2\nethernets:\n  eth0:\n    dhcp4: true\n    addresses: [%s/32]\n", t.SlotAddress())
}

// Guests returns the resources that settle each guest of the list, in order: its instance, its
// start, and its setup. It fails where a guest's own values give no tunnel, before it reads Incus:
// every guest needs a slot. A guest of ws's that the list lacks gets a warning on out, and no deletion.
func Guests(ctx context.Context, m effect.Machine, guests []values.Guest, out effect.Out) ([]resource.Resource, error) {
	tunnels := make([]values.GuestTunnel, len(guests))
	gpus := make([]bool, len(guests))
	for i, g := range guests {
		_, own, err := g.Read(m)
		if err != nil {
			return nil, err
		}
		if tunnels[i], err = g.Tunnel(own); err != nil {
			return nil, err
		}
		gpus[i] = own.Nvidia.GPU
	}
	alias, err := Image{M: m}.Alias(ctx)
	if err != nil {
		return nil, err
	}
	held, err := Instances(ctx, m)
	if err != nil {
		return nil, err
	}
	for _, name := range Strays(held, guests) {
		out.Warn("the guest %s is not in WS_INCUS_GUESTS: ws deletes none, so remove it by hand with ws ubuntu host guests remove %s", name, name)
	}
	var steps []resource.Resource
	for i, g := range guests {
		steps = append(steps, Guest(m, tunnels[i], gpus[i], alias), Running{M: m, Name: g.Name})
		steps = append(steps, SetUp(m, g, false, false, out)...)
	}
	return steps, nil
}

// guestValuesPath is where a guest reads its values, as a host does.
const guestValuesPath = "/etc/ws/values.yml"

// SetUp returns the resources that set a running guest up: ws's binary and the guest's own values
// pushed, then the guest's setup. The values file is the guest's own on the host, and is pushed
// where the host holds one. The setup is due where the guest's own plan is, or always with force,
// and secrets adds the task that clones the secrets. What the setup prints goes to out.
func SetUp(m effect.Machine, g values.Guest, secrets, force bool, out effect.Out) []resource.Resource {
	return []resource.Resource{
		Pushed{M: m, Guest: g.Name, From: wsInGuest, To: wsInGuest, Mode: "0755"},
		Pushed{M: m, Guest: g.Name, From: g.File(), To: guestValuesPath, Mode: "0600", Optional: true},
		Setup{M: m, Name: g.Name, Secrets: secrets, Force: force, Out: out},
	}
}
