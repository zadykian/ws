package network

import (
	"encoding/json"
	"slices"
	"testing"
)

const ownAddress = "10.77.0.1/30"

func parse(t *testing.T, links, routes string) ([]Link, []Route) {
	var l []Link
	var r []Route
	if err := json.Unmarshal([]byte(links), &l); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(routes), &r); err != nil {
		t.Fatal(err)
	}
	return l, r
}

func TestUplinkProblems(t *testing.T) {
	tests := []struct {
		name, links, routes string
		want                []string
	}{
		{"a port with no route and no lease", `[{"ifname":"enp5s0","addr_info":[]}]`, `[]`, nil},
		{"the direct link's own address", `[{"ifname":"enp5s0","addr_info":[{"family":"inet","local":"10.77.0.1","prefixlen":30,"scope":"global"}]}]`, `[]`, nil},
		{"a static address outside the subnet", `[{"ifname":"enp5s0","addr_info":[{"family":"inet","local":"192.168.1.5","prefixlen":24,"scope":"global"}]}]`, `[]`, nil},
		{"a lease inside the subnet", `[{"ifname":"enp5s0","addr_info":[{"family":"inet","local":"10.77.0.2","prefixlen":30,"scope":"global","dynamic":true}]}]`, `[]`, nil},
		{"a lease outside the subnet", `[{"ifname":"enp5s0","addr_info":[{"family":"inet","local":"192.168.1.5","prefixlen":24,"scope":"global","dynamic":true}]}]`,
			`[]`, []string{"enp5s0 holds 192.168.1.5/24 from DHCP, outside 10.77.0.0/30"}},
		{"a dynamic IPv6 address", `[{"ifname":"enp5s0","addr_info":[{"family":"inet6","local":"fd00::5","prefixlen":64,"scope":"global","dynamic":true}]}]`, `[]`, nil},
		{"a link-local lease", `[{"ifname":"enp5s0","addr_info":[{"family":"inet","local":"169.254.1.1","prefixlen":16,"scope":"link","dynamic":true}]}]`, `[]`, nil},
		{"a default route", `[{"ifname":"enp5s0","addr_info":[]}]`, `[{"dev":"enp5s0"}]`,
			[]string{"enp5s0 carries a default route, in table main"}},
		{"a route of another interface", `[{"ifname":"enp5s0","addr_info":[]}]`, `[{"dev":"wg0","table":"51820"}]`, nil},
		{"routes in two tables", `[{"ifname":"enp5s0","addr_info":[]}]`, `[{"dev":"enp5s0","table":"main"},{"dev":"enp5s0","table":"100"},{"dev":"enp5s0","table":"100"}]`,
			[]string{"enp5s0 carries a default route, in table 100, table main"}},
		{"a next hop", `[{"ifname":"enp5s0","addr_info":[]}]`, `[{"dev":"eth9","nexthops":[{"dev":"enp5s0"}]}]`,
			[]string{"enp5s0 carries a default route, in table main"}},
		{"a route and a lease", `[{"ifname":"enp5s0","addr_info":[{"family":"inet","local":"192.168.1.5","prefixlen":24,"scope":"global","dynamic":true}]}]`, `[{"dev":"enp5s0"}]`,
			[]string{"enp5s0 carries a default route, in table main", "enp5s0 holds 192.168.1.5/24 from DHCP, outside 10.77.0.0/30"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links, routes := parse(t, tt.links, tt.routes)
			got, err := UplinkProblems(links, routes, ownAddress)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUplinkRefusesWhatItCannotRead(t *testing.T) {
	links, routes := parse(t, `[{"ifname":"enp5s0","addr_info":[{"family":"inet","local":"nope","prefixlen":24,"scope":"global","dynamic":true}]}]`, `[]`)
	if _, err := UplinkProblems(links, routes, ownAddress); err == nil {
		t.Error("an unreadable lease: want an error")
	}
	if _, err := UplinkProblems(links, routes, "10.77.0.1"); err == nil {
		t.Error("an address with no prefix: want an error")
	}
}
