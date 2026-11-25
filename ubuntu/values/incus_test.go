package values

import (
	"slices"
	"strings"
	"testing"
)

func TestIncusDefaults(t *testing.T) {
	h, err := ParseHost(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := Incus{BridgeAddress: "10.100.0.1/24", BridgeDNS: []string{"9.9.9.9", "1.1.1.1"}}
	if h.Incus.PoolSource != want.PoolSource || h.Incus.BridgeAddress != want.BridgeAddress || !slices.Equal(h.Incus.BridgeDNS, want.BridgeDNS) {
		t.Errorf("the defaults are %+v, want %+v: no partition is the host's to guess", h.Incus, want)
	}
}

func TestParseHostRefusesABadIncusValue(t *testing.T) {
	for data, want := range map[string]string{
		"WS_INCUS_POOL_SOURCE: sdb1\n":                                                "WS_INCUS_POOL_SOURCE",
		"WS_INCUS_POOL_SOURCE: /dev/sdb1 x\n":                                         "WS_INCUS_POOL_SOURCE",
		"WS_INCUS_POOL_SOURCE: /home/x\n":                                             "WS_INCUS_POOL_SOURCE",
		"WS_INCUS_BRIDGE_ADDRESS: 10.100.0.0/24\n":                                    "WS_INCUS_BRIDGE_ADDRESS: 10.100.0.0 is the address of the subnet",
		"WS_INCUS_BRIDGE_ADDRESS: 10.100.0.1\n":                                       "WS_INCUS_BRIDGE_ADDRESS",
		"WS_INCUS_BRIDGE_ADDRESS: 10.100.0.1/024\n":                                   "WS_INCUS_BRIDGE_ADDRESS",
		"WS_INCUS_BRIDGE_ADDRESS: 198.18.0.1/24\n":                                    "outside the private ranges",
		"WS_INCUS_BRIDGE_ADDRESS: 10.0.0.1/7\n":                                       "outside the private ranges",
		"WS_INCUS_BRIDGE_DNS: [192.168.1.1]\n":                                        "a private address",
		"WS_INCUS_BRIDGE_DNS: [100.64.0.1]\n":                                         "a private address",
		"WS_INCUS_BRIDGE_ADDRESS: 10.99.4.1/24\n":                                     "overlaps 10.99.0.0/20",
		"WS_INCUS_BRIDGE_ADDRESS: 10.98.0.1/15\n":                                     "overlaps 10.99.0.0/20",
		"WS_INCUS_BRIDGE_DNS: []\n":                                                   "WS_INCUS_BRIDGE_DNS",
		"WS_INCUS_BRIDGE_DNS: [9.9.9.9, dns.example]\n":                               "WS_INCUS_BRIDGE_DNS",
		"WS_INCUS_BRIDGE_DNS: [\"2620:fe::fe\"]\n":                                    "WS_INCUS_BRIDGE_DNS",
		"WS_INCUS_BRIDGE_DNS: [9.9.9.9, 9.9.9.9]\n":                                   "WS_INCUS_BRIDGE_DNS",
		"WS_INCUS_BRIDGE_DNS: [\"9.9.9.9,10.0.0.1\"]\n":                               "WS_INCUS_BRIDGE_DNS",
		"WS_INCUS_BRIDGE_DNS: [10.100.0.1]\n":                                         "WS_INCUS_BRIDGE_DNS",
		"WS_INCUS_BRIDGE_ADDRESS: 10.100.0.1/24\nWS_INCUS_BRIDGE_DNS: [10.100.0.1]\n": "WS_INCUS_BRIDGE_DNS",
	} {
		if _, err := ParseHost([]byte(data)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("ParseHost(%q) = %v, want %q", data, err, want)
		}
	}
}

func TestParseHostTakesAnIncusPartition(t *testing.T) {
	data := "WS_INCUS_POOL_SOURCE: /dev/disk/by-partuuid/0a1b\nWS_INCUS_BRIDGE_ADDRESS: 10.100.4.1/22\nWS_INCUS_BRIDGE_DNS: [8.8.8.8]\n"
	h, err := ParseHost([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if h.Incus.PoolSource != "/dev/disk/by-partuuid/0a1b" || h.Incus.BridgeAddress != "10.100.4.1/22" || !slices.Equal(h.Incus.BridgeDNS, []string{"8.8.8.8"}) {
		t.Errorf("Incus = %+v", h.Incus)
	}
}

func TestIncusKeysAreTheHostsAlone(t *testing.T) {
	for _, k := range Keys() {
		if strings.HasPrefix(k.Name, "WS_INCUS_") {
			t.Errorf("the guest holds %s", k.Name)
		}
	}
	names := map[string]Kind{}
	for _, k := range HostKeys() {
		names[k.Name] = k.Kind
	}
	for _, name := range []string{"WS_INCUS_POOL_SOURCE", "WS_INCUS_BRIDGE_ADDRESS", "WS_INCUS_BRIDGE_DNS"} {
		if _, ok := names[name]; !ok {
			t.Errorf("the host's keys lack %s", name)
		}
	}
}

func TestIncusRangesAreWhatATemplateNames(t *testing.T) {
	i := Incus{BridgeAddress: "10.100.4.9/22"}
	if got := i.BridgeSubnet(); got != "10.100.4.0/22" {
		t.Errorf("BridgeSubnet = %s, want the subnet of the address", got)
	}
	if got := i.TunnelSlots(); got != "10.99.0.0/20" {
		t.Errorf("TunnelSlots = %s", got)
	}
}
