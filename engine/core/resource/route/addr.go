package route

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/zadykian/ws/engine/core/resource"
)

// Addr stops the run where Iface lacks the address Want, with its prefix length. The run cannot
// move an interface's address, so it plans a change that fails with Hint. A missing interface
// holds no address.
type Addr struct {
	Iface string
	Want  netip.Prefix
	Hint  string
	// Held returns the interface's IPv4 addresses; nil reads the machine's, by Addrs.
	Held func(iface string) ([]netip.Prefix, error)
}

// moves is the address that Iface would have to take.
type moves struct {
	a     Addr
	holds string
}

func (c moves) String() string {
	return fmt.Sprintf("%s holds %s, not %s", c.a.Iface, c.holds, c.a.Want)
}

// Apply fails: the run cannot move the address.
func (c moves) Apply(context.Context) error { return fmt.Errorf("%s: %s", c, c.a.Hint) }

// Plan reads the interface's IPv4 addresses.
func (a Addr) Plan(context.Context) ([]resource.Change, error) {
	held := a.Held
	if held == nil {
		held = Addrs
	}
	got, err := held(a.Iface)
	if err != nil {
		return nil, err
	}
	var text []string
	for _, p := range got {
		if p == a.Want {
			return nil, nil
		}
		text = append(text, p.String())
	}
	holds := "no address"
	if len(text) > 0 {
		holds = strings.Join(text, ", ")
	}
	return []resource.Change{moves{a, holds}}, nil
}

// Addrs returns the IPv4 addresses of the interface iface, each with its prefix length, or none
// where no interface has that name.
func Addrs(iface string) ([]netip.Prefix, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, ifc := range ifaces {
		if ifc.Name != iface {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			return nil, err
		}
		var held []netip.Prefix
		for _, a := range addrs {
			if p, err := netip.ParsePrefix(a.String()); err == nil && p.Addr().Is4() {
				held = append(held, p)
			}
		}
		return held, nil
	}
	return nil, nil
}
