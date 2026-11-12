package setup

import "testing"

func TestServiceListed(t *testing.T) {
	list := "An asterisk (*) denotes that a network service is disabled.\nWi-Fi\n*USB 10/100 LAN\nCable\n"
	for _, tc := range []struct {
		service         string
		listed, disable bool
	}{
		{"Cable", true, false}, {"USB 10/100 LAN", true, true}, {"Ethernet", false, false},
	} {
		listed, disabled := serviceListed(list, tc.service)
		if listed != tc.listed || disabled != tc.disable {
			t.Errorf("serviceListed(%q) = %v, %v; want %v, %v", tc.service, listed, disabled, tc.listed, tc.disable)
		}
	}
}

func TestCableOther(t *testing.T) {
	const set = "Manual Configuration\nIP address: 10.77.0.2\nSubnet mask: 255.255.255.252\nRouter: (null)\n"
	for name, tc := range map[string]struct{ info, want string }{
		"set":       {set, ""},
		"no lines":  {"", "nothing"},
		"left out":  {"Manual Configuration\n", ""},
		"dhcp":      {"DHCP Configuration\nIP address: 192.168.1.5\nSubnet mask: 255.255.255.0\nRouter: 192.168.1.1\n", "DHCP Configuration; IP address: 192.168.1.5; Subnet mask: 255.255.255.0; Router: 192.168.1.1"},
		"our alias": {"Manual Configuration\nIP address: 10.1.1.1\nIP address: 10.77.0.2\n", ""},
		"a router":  {"Manual Configuration\nIP address: 10.77.0.2\r\nRouter: 10.77.0.1\n", "Router: 10.77.0.1"},
	} {
		if got := cableOther(tc.info); got != tc.want {
			t.Errorf("%s: cableOther = %q, want %q", name, got, tc.want)
		}
	}
}
