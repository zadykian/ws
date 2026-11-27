package values

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/cablenet"
)

func TestCablePortRefusesWhatJSONEscapes(t *testing.T) {
	for _, port := range []string{"../x", "a b", "a\tb", "a\"b", `a\b`, "a<b", "a>b", "a&b", "a\x01b", "a\x7fb"} {
		err := Cable{Port: port}.check()
		if err == nil || !strings.Contains(err.Error(), "is not an interface's name") {
			t.Errorf("port %q: check = %v, want a refusal", port, err)
		}
	}
	for _, port := range []string{"enp5s0", "enx00e04c680001", "eth0.10", "wan-1_a"} {
		if err := (Cable{Port: port}).check(); err != nil {
			t.Errorf("port %q: check = %v, want none", port, err)
		}
	}
}

func TestEmptyCablePortChecksNothing(t *testing.T) {
	if err := (Cable{}).check(); err != nil {
		t.Errorf("check = %v, want none without a port", err)
	}
}

func TestParseHostRefusesABadCable(t *testing.T) {
	if _, err := ParseHost([]byte("WS_CABLE_PORT: a b\n")); err == nil || !strings.Contains(err.Error(), `WS_CABLE_PORT "a b" is not an interface's name`) {
		t.Errorf("ParseHost = %v, want the port refused", err)
	}
}

// The cable's /30 is fixed for both sides, so no key sets the server's end.
func TestParseHostKnowsNoCableAddress(t *testing.T) {
	_, err := ParseHost([]byte("WS_CABLE_PORT: enp5s0\nWS_CABLE_ADDRESS: 10.77.0.1/30\n"))
	if err == nil || !strings.Contains(err.Error(), "WS_CABLE_ADDRESS") {
		t.Errorf("ParseHost = %v, want the key refused", err)
	}
}

func TestCableAddressIsTheServersEnd(t *testing.T) {
	if got := (Cable{}).Address().String(); got != "10.77.0.1/30" {
		t.Errorf("Address = %s, want 10.77.0.1/30", got)
	}
}

func TestSetRefusesABadCablePort(t *testing.T) {
	if _, err := SetHost([]byte("---\n"), "WS_CABLE_PORT", "a&b"); err == nil || !strings.Contains(err.Error(), "is not an interface's name") {
		t.Errorf("SetHost = %v, want the port refused", err)
	}
}

// A guest reaches the host's addresses that the egress ACL does not drop, so the cable's must be
// among the private ranges.
func TestCableLiesInThePrivateRanges(t *testing.T) {
	if !inPrivateRanges(cablenet.Net) {
		t.Errorf("%s lies outside the private ranges, which the guests' ACL drops", cablenet.Net)
	}
}
