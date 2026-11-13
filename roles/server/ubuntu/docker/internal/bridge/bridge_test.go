package bridge

import (
	"net/netip"
	"testing"
)

func TestAddrsOfReadsTheIPv4AddressesOfAnInterface(t *testing.T) {
	got, err := addrsOf("lo")
	if err != nil || len(got) == 0 || got[0] != netip.MustParsePrefix("127.0.0.1/8") {
		t.Errorf("addrsOf(lo) = %v, %v; want 127.0.0.1/8 first", got, err)
	}
	if got, err := addrsOf("no-such-ifc0"); err != nil || got != nil {
		t.Errorf("addrsOf(missing) = %v, %v; want none, as docker0 before Docker's first start", got, err)
	}
}
