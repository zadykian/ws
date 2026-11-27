package values

import (
	"strings"
	"testing"
)

func TestCablePortRefusesWhatJSONEscapes(t *testing.T) {
	for _, port := range []string{"../x", "a b", "a\tb", "a\"b", `a\b`, "a<b", "a>b", "a&b", "a\x01b", "a\x7fb"} {
		err := Cable{Port: port, Address: "10.77.0.1/30"}.check()
		if err == nil || !strings.Contains(err.Error(), "is not an interface's name") {
			t.Errorf("port %q: check = %v, want a refusal", port, err)
		}
	}
	for _, port := range []string{"enp5s0", "enx00e04c680001", "eth0.10", "wan-1_a"} {
		if err := (Cable{Port: port, Address: "10.77.0.1/30"}).check(); err != nil {
			t.Errorf("port %q: check = %v, want none", port, err)
		}
	}
}

func TestCableAddressRefusedWithAPort(t *testing.T) {
	err := Cable{Port: "enp5s0", Address: "10.77.0.3/30"}.check()
	if err == nil || err.Error() != "WS_CABLE_ADDRESS: 10.77.0.3 is the broadcast address of 10.77.0.0/30, not a host's" {
		t.Errorf("check = %v, want the address's problem", err)
	}
}

func TestEmptyCablePortChecksNothing(t *testing.T) {
	if err := (Cable{Address: "nonsense"}).check(); err != nil {
		t.Errorf("check = %v, want none without a port", err)
	}
}

func TestParseHostRefusesABadCable(t *testing.T) {
	for data, want := range map[string]string{
		"WS_CABLE_PORT: a b\n": `WS_CABLE_PORT "a b" is not an interface's name`,
		"WS_CABLE_PORT: enp5s0\nWS_CABLE_ADDRESS: 10.77.0.0/30\n": "WS_CABLE_ADDRESS: 10.77.0.0 is the address of the subnet",
	} {
		if _, err := ParseHost([]byte(data)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("ParseHost(%q) = %v, want %q", data, err, want)
		}
	}
}

func TestSetRefusesABadCablePort(t *testing.T) {
	if _, err := SetHost([]byte("---\n"), "WS_CABLE_PORT", "a&b"); err == nil || !strings.Contains(err.Error(), "is not an interface's name") {
		t.Errorf("SetHost = %v, want the port refused", err)
	}
}

// A guest reaches the host's addresses that the egress ACL does not drop, so the cable's must be
// among the private ranges.
func TestCableAddressRefusedOutsideThePrivateRanges(t *testing.T) {
	for _, address := range []string{"198.18.0.1/30", "10.77.0.1/030"} {
		err := Cable{Port: "enp5s0", Address: address}.check()
		if err == nil || !strings.Contains(err.Error(), "WS_CABLE_ADDRESS") {
			t.Errorf("address %q: check = %v, want a refusal", address, err)
		}
	}
}
