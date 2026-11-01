package network

import (
	"fmt"
	"net/netip"
	"slices"
	"strings"
)

// Route is one default route of `ip -j route show default table all`.
type Route struct {
	Dev      string `json:"dev"`
	Table    string `json:"table"`
	Nexthops []struct {
		Dev string `json:"dev"`
	} `json:"nexthops"`
}

// Link is an interface of `ip -j address show dev NAME`.
type Link struct {
	Name     string   `json:"ifname"`
	Altnames []string `json:"altnames"`
	Address  string   `json:"address"`
	// Permaddr is the permanent address, which ip names only where it differs from Address.
	Permaddr string `json:"permaddr"`
	Held     []Held `json:"addr_info"`
}

// Held is an address on an interface.
type Held struct {
	Family    string `json:"family"`
	Local     string `json:"local"`
	Prefixlen int    `json:"prefixlen"`
	Dynamic   bool   `json:"dynamic"`
	Scope     string `json:"scope"`
}

// devices returns the interfaces a route goes out of, each of its next hops' too.
func (r Route) devices() []string {
	var devs []string
	if r.Dev != "" {
		devs = append(devs, r.Dev)
	}
	for _, hop := range r.Nexthops {
		if hop.Dev != "" {
			devs = append(devs, hop.Dev)
		}
	}
	return devs
}

// UplinkProblems returns why the interfaces of links look like an uplink. routes are the default
// routes of every table, for both families. An uplink carries a default route, in any table, or
// holds an IPv4 address from DHCP, which ip calls dynamic, outside the subnet of address, the
// direct link's. A static address outside it, as an earlier run's address, does not count: DHCP's
// are the ones the file takes away.
func UplinkProblems(links []Link, routes []Route, address string) ([]string, error) {
	own, err := netip.ParsePrefix(address)
	if err != nil {
		return nil, fmt.Errorf("the direct link's address: %w", err)
	}
	subnet := own.Masked()
	var problems []string
	for _, link := range links {
		var tables []string
		for _, r := range routes {
			table := r.Table
			if table == "" {
				table = "main"
			}
			if slices.Contains(r.devices(), link.Name) && !slices.Contains(tables, table) {
				tables = append(tables, table)
			}
		}
		slices.Sort(tables)
		for i, table := range tables {
			tables[i] = "table " + table
		}
		if len(tables) > 0 {
			problems = append(problems, fmt.Sprintf("%s carries a default route, in %s", link.Name, strings.Join(tables, ", ")))
		}
		for _, held := range link.Held {
			if held.Family != "inet" || !held.Dynamic || held.Scope != "global" {
				continue
			}
			ip, err := netip.ParseAddr(held.Local)
			if err != nil {
				return nil, fmt.Errorf("%s holds the address %q: %w", link.Name, held.Local, err)
			}
			if !subnet.Contains(ip) {
				problems = append(problems, fmt.Sprintf("%s holds %s/%d from DHCP, outside %s", link.Name, ip, held.Prefixlen, subnet))
			}
		}
	}
	return problems, nil
}
