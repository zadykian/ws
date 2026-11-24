package cable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/ubuntu/roles/cable/internal/netplan"
)

// readPort reads the port and the default routes, refuses an uplink, and returns the port as
// netplan's match sees it. ip's default_ipv4 would name the interface of the route to one address.
// A VPN's 0.0.0.0/1 and 128.0.0.0/1 take that route from the uplink. So each table's default
// routes name the uplink, whatever a VPN adds. The file turns DHCP off on the port: on an uplink, that
// would cut the machine off. A VPN that takes the default route still leaves the uplink its lease.
func readPort(ctx context.Context, m effect.Machine, name, address string) (netplan.Port, error) {
	var routes []Route
	for _, family := range []string{"-4", "-6"} {
		var found []Route
		if err := readJSON(ctx, m, &found, "ip", "-j", family, "route", "show", "default", "table", "all"); err != nil {
			return netplan.Port{}, err
		}
		routes = append(routes, found...)
	}
	var links []Link
	if err := readJSON(ctx, m, &links, "ip", "-j", "address", "show", "dev", name); err != nil {
		return netplan.Port{}, err
	}
	if len(links) == 0 {
		return netplan.Port{}, fmt.Errorf("ip lists no interface %s", name)
	}
	problems, err := UplinkProblems(links, routes, address)
	if err != nil {
		return netplan.Port{}, err
	}
	if len(problems) > 0 {
		return netplan.Port{}, fmt.Errorf("WS_CABLE_PORT names the port of the Mac's cable, not an uplink, yet %s", strings.Join(problems, " and "))
	}
	// networkd's Driver= reads udev's ID_NET_DRIVER. A module fact lacks a driver built into the
	// kernel, as virtio_net may be.
	driver, err := m.Run(ctx, "udevadm", "info", "--query=property", "--property=ID_NET_DRIVER", "--value", "/sys/class/net/"+name)
	if err != nil {
		return netplan.Port{}, err
	}
	link := links[0]
	mac := link.Permaddr // ip names the permanent address only where it differs from the address
	if mac == "" {
		mac = link.Address
	}
	if mac == "" {
		return netplan.Port{}, errors.New("ip shows no address for " + name)
	}
	return netplan.Port{Name: name, Altnames: link.Altnames, MAC: mac, Driver: strings.TrimSpace(string(driver))}, nil
}

// readJSON runs a command and reads its standard output as JSON.
func readJSON(ctx context.Context, m effect.Machine, into any, name string, args ...string) error {
	out, err := m.Run(ctx, name, args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(out, into); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
