package incus

import (
	"context"
	"fmt"
	"net/netip"
	"slices"
	"strconv"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/internal/slot"
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

// Guest is the instance of a guest of the host's list. It takes the profile ws, its limits, its NIC
// at its address on the port that the table of sources names, and its disk with its quota. It
// takes the host's NVIDIA card as well where its WS_NVIDIA_GPU is on, and none otherwise.
// Each device is whole, as a device of an instance replaces the profile's of its name. The seed's
// network is a key of the create alone, as cloud-init reads the seed once. An instance that holds
// no profile ws is refused, as that instance is no guest of ws's.
func Guest(m effect.Machine, g values.Guest, own values.Values, alias string) object.Object {
	return object.Object{
		M: m, Kind: guestKind(alias), Name: g.Name,
		Want: map[string]any{
			"description": "ws's guest " + g.Name,
			"profiles":    []any{ProfileName},
			"config":      map[string]any{"limits.cpu": strconv.Itoa(g.CPU), "limits.memory": g.Memory},
			"devices": map[string]any{
				"eth0": nicDevice(nicKeys(g, own)),
				"root": rootDevice(g.Disk),
				"gpu":  gpuDevice(own.Nvidia.GPU),
			},
		},
		First: map[string]any{"config": map[string]any{"cloud-init.network-config": seedNetwork(own)}},
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
// route, where the guest's values name a slot. Incus then sends that address to this NIC, which is
// where the host's tunnel for the guest delivers.
func nicKeys(g values.Guest, own values.Values) map[string]any {
	keys := map[string]any{"host_name": g.Port(), "ipv4.address": g.Address}
	if t, err := g.Tunnel(own); err == nil {
		keys["ipv4.routes"] = t.SlotAddress().String() + "/32"
	}
	return keys
}

// seedNetwork is the guest's network-config. It holds DHCP on eth0, which gives the NIC's address,
// and the slot's address where the guest's own values name a slot. The tunnel ends on the host,
// which routes that address to the guest. So the guest holds it as a host route.
func seedNetwork(v values.Values) string {
	text := "version: 2\nethernets:\n  eth0:\n    dhcp4: true\n"
	if p, err := netip.ParsePrefix(v.Tunnel.Address); err == nil {
		if s, ok := slot.Find(p.Addr()); ok && s != 0 {
			text += fmt.Sprintf("    addresses: [%s/32]\n", p.Addr())
		}
	}
	return text
}

// Guests returns the resources that settle each guest of the list, in order: its instance, its
// start, and its setup. A guest of ws's that the list lacks gets a warning on out, and no deletion.
func Guests(ctx context.Context, m effect.Machine, guests []values.Guest, out effect.Out) ([]resource.Resource, error) {
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
	for _, g := range guests {
		_, own, err := g.Read(m)
		if err != nil {
			return nil, err
		}
		steps = append(steps, Guest(m, g, own, alias), Running{M: m, Name: g.Name})
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
