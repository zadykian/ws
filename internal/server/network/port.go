package network

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// checkPort reads the port and the default routes, refuses an uplink, and returns the port as
// netplan's match sees it. ip's default_ipv4 would name the interface of the route to one address.
// A VPN's 0.0.0.0/1 and 128.0.0.0/1 take that route from the uplink. So each table's default
// routes name the uplink, whatever a VPN adds. The file turns DHCP off on the port: on an uplink, that
// would cut the machine off. A VPN that takes the default route still leaves the uplink its lease.
func (d direct) checkPort(ctx context.Context) (Port, error) {
	var routes []Route
	for _, family := range []string{"-4", "-6"} {
		var found []Route
		if err := d.readJSON(ctx, &found, "ip", "-j", family, "route", "show", "default", "table", "all"); err != nil {
			return Port{}, err
		}
		routes = append(routes, found...)
	}
	var links []Link
	if err := d.readJSON(ctx, &links, "ip", "-j", "address", "show", "dev", d.Interface); err != nil {
		return Port{}, err
	}
	if len(links) == 0 {
		return Port{}, fmt.Errorf("ip lists no interface %s", d.Interface)
	}
	problems, err := UplinkProblems(links, routes, d.Address)
	if err != nil {
		return Port{}, err
	}
	if len(problems) > 0 {
		return Port{}, fmt.Errorf("network_direct_link_interface names the port of the Mac's cable, not an uplink, yet %s", strings.Join(problems, " and "))
	}
	// networkd's Driver= reads udev's ID_NET_DRIVER. A module fact lacks a driver built into the
	// kernel, as virtio_net may be.
	driver, err := d.M.Run(ctx, "udevadm", "info", "--query=property", "--property=ID_NET_DRIVER", "--value", "/sys/class/net/"+d.Interface)
	if err != nil {
		return Port{}, err
	}
	link := links[0]
	mac := link.Permaddr // ip names the permanent address only where it differs from the address
	if mac == "" {
		mac = link.Address
	}
	if mac == "" {
		return Port{}, errors.New("ip shows no address for " + d.Interface)
	}
	return Port{Name: d.Interface, Altnames: link.Altnames, MAC: mac, Driver: strings.TrimSpace(string(driver))}, nil
}
