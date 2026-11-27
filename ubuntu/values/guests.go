package values

import (
	"errors"
	"fmt"
	"io/fs"
	"net/netip"
	"path"
	"regexp"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// GuestsDir holds a guest's own values on the host, in a folder of its name.
const GuestsDir = "/etc/ws/guests"

// portPrefix starts the host's name for a guest's NIC. The name is 15 bytes at most, so a guest's
// name takes 12.
const portPrefix = "ws-"

var (
	guestName = regexp.MustCompile(`^[a-z]([a-z0-9-]{0,10}[a-z0-9])?$`)
	// guestSize is a size that Incus reads as a whole number of binary units.
	guestSize = regexp.MustCompile(`^[1-9][0-9]*(MiB|GiB|TiB)$`)
)

// Guest is a guest that the host runs, with the limits that Incus holds it to. Its other values
// are the guest's own, in its file.
type Guest struct {
	// Name is the guest's name and its hostname. It reaches Incus's commands, a path, and the
	// secrets' profile, so it holds lowercase letters, digits and dashes, and starts with a letter.
	Name string `yaml:"name"`
	// Address is the guest's address on the bridge: its lease and the source it may send from.
	Address string `yaml:"address"`
	// CPU is the number of cores the guest may use.
	CPU int `yaml:"cpu"`
	// Memory is the guest's memory limit, as 8GiB, and Disk its root disk's quota, as 200GiB.
	Memory string `yaml:"memory"`
	Disk   string `yaml:"disk"`

	// tunnel is the guest's tunnel, and noTunnel the reason it has none, as LoadHost works them out
	// from the guest's own values. Both are unset on a guest that no load has resolved.
	tunnel   *GuestTunnel
	noTunnel error
}

// Source is an address that a guest may send from, with the name of the port that carries it.
type Source struct{ Address, Port string }

// Sources are the addresses that the guest may send from: its own on the bridge, and its slot's
// address where it has a slot. It holds that one on eth0, and its peers' replies leave from it.
func (g Guest) Sources() []Source {
	sources := []Source{{g.Address, g.Port()}}
	if g.tunnel != nil {
		sources = append(sources, Source{g.tunnel.SlotAddress().String(), g.Port()})
	}
	return sources
}

// Link is the name of the guest's tunnel interface on the host, or empty where it has no slot.
func (g Guest) Link() string {
	if g.tunnel == nil {
		return ""
	}
	return g.tunnel.Link()
}

// SlotNet is the subnet of the guest's slot, where its tunnel's peers are, or empty where it has no
// slot. A source of no guest's may leave the guest's port toward it alone: a reply that the guest
// forwards to a peer.
func (g Guest) SlotNet() string {
	if g.tunnel == nil {
		return ""
	}
	return g.tunnel.Slot.Net().String()
}

// Mark is the firewall mark that what comes in by the guest's NIC takes, and that the way out by
// its tunnel interface asks for. Zero where the guest has no slot.
func (g Guest) Mark() int {
	if g.tunnel == nil {
		return 0
	}
	return g.tunnel.Mark()
}

// Load returns the guest resolved: its own values read on m, and from them its tunnel or the reason
// it has none. A file that fails to read leaves the reason, not an error, so that a host runs with
// one bad guest file where a role does not need its tunnel.
func (g Guest) Load(m effect.Machine) Guest {
	_, own, err := g.Read(m)
	if err != nil {
		g.tunnel, g.noTunnel = nil, err
		return g
	}
	return g.Resolve(own)
}

// Resolve returns the guest with the tunnel that its own values give, or the reason they give none.
func (g Guest) Resolve(own Values) Guest {
	t, err := g.Tunnel(own)
	if err != nil {
		g.tunnel, g.noTunnel = nil, err
		return g
	}
	g.tunnel, g.noTunnel = &t, nil
	return g
}

// Resolved returns the guest's tunnel, or the reason it has none: its own values did not read or name
// no slot, or no load has resolved the guest.
func (g Guest) Resolved() (GuestTunnel, error) {
	switch {
	case g.tunnel != nil:
		return *g.tunnel, nil
	case g.noTunnel != nil:
		return GuestTunnel{}, g.noTunnel
	}
	return GuestTunnel{}, fmt.Errorf("%s: its own values are not loaded", g.Name)
}

// File is the guest's own values on the host, which the host pushes into the guest.
func (g Guest) File() string { return path.Join(GuestsDir, g.Name, "values.yml") }

// Port is the name of the guest's NIC on the host, which the table of sources names.
func (g Guest) Port() string { return portPrefix + g.Name }

// errSelfContained refuses a guest whose values turn the mode on. Its tunnel would end at a LAN
// address, which a guest under a host lacks, and the host gives it SigNoz.
var errSelfContained = errors.New("WS_SELF_CONTAINED is on, which a guest under a host cannot run: the host ends its tunnel and runs SigNoz")

// Read returns the guest's own values, as its file holds them on the host, and the file's bytes.
// A missing file gives no bytes and the defaults. A file that does not parse fails, naming it, and
// so does one that turns the self-contained mode on.
func (g Guest) Read(m effect.Machine) ([]byte, Values, error) {
	data, err := m.ReadFile(g.File())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, Default(), nil
	}
	if err != nil {
		return nil, Values{}, err
	}
	v, err := Parse(data)
	if err != nil {
		return nil, Values{}, fmt.Errorf("%s: %w", g.File(), err)
	}
	if v.SelfContained {
		return nil, Values{}, fmt.Errorf("%s: %w", g.File(), errSelfContained)
	}
	return data, v, nil
}

// guestsCheck refuses a list that no guest could run by: each guest's fields, and no name or
// address twice. bridge is the bridge's address with its prefix length, whose subnet the guests share.
func guestsCheck(guests []Guest, bridge netip.Prefix) error {
	names, addrs := map[string]bool{}, map[netip.Addr]bool{}
	for i, g := range guests {
		if err := g.check(bridge); err != nil {
			return fmt.Errorf("WS_INCUS_GUESTS, guest %d: %w", i+1, err)
		}
		ip := netip.MustParseAddr(g.Address)
		switch {
		case names[g.Name]:
			return fmt.Errorf("WS_INCUS_GUESTS holds the name %s twice", g.Name)
		case addrs[ip]:
			return fmt.Errorf("WS_INCUS_GUESTS holds the address %s twice", g.Address)
		}
		names[g.Name], addrs[ip] = true, true
	}
	return nil
}

// check refuses a guest that Incus could not run. It names the field.
func (g Guest) check(bridge netip.Prefix) error {
	var problems []string
	if !guestName.MatchString(g.Name) {
		problems = append(problems, fmt.Sprintf("name %q is not 1 to 12 lowercase letters, digits and dashes, starting with a letter and not ending in a dash.", g.Name))
	}
	if p := g.addressProblem(bridge); p != "" {
		problems = append(problems, p)
	}
	if g.CPU < 1 {
		problems = append(problems, fmt.Sprintf("cpu is %d, not a number of cores from 1.", g.CPU))
	}
	for _, f := range []struct{ field, size string }{{"memory", g.Memory}, {"disk", g.Disk}} {
		if !guestSize.MatchString(f.size) {
			problems = append(problems, fmt.Sprintf("%s is %q, not a whole number of MiB, GiB or TiB, as 8GiB.", f.field, f.size))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return errors.New(g.label() + strings.Join(problems, " "))
}

// label starts a guest's problems with the guest, where it has a name.
func (g Guest) label() string {
	if g.Name == "" {
		return ""
	}
	return g.Name + ": "
}

// addressProblem says why the address is no guest's on the bridge, or returns nothing. The bridge
// holds the first address, and the broadcast and the network's own are no host's.
func (g Guest) addressProblem(bridge netip.Prefix) string {
	ip, err := netip.ParseAddr(g.Address)
	switch {
	case err != nil || !ip.Is4():
		return fmt.Sprintf("address %q is not an IPv4 address, as 10.100.0.10.", g.Address)
	case !bridge.Masked().Contains(ip):
		return fmt.Sprintf("address %s is not in the bridge's subnet %s.", ip, bridge.Masked())
	case ip == bridge.Addr():
		return fmt.Sprintf("address %s is the bridge's own address.", ip)
	case !isHost(ip, bridge.Masked()):
		return fmt.Sprintf("address %s is not a host's address in %s.", ip, bridge.Masked())
	}
	return ""
}
