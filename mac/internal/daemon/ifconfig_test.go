package daemon

import (
	"net/netip"
	"testing"

	"github.com/zadykian/ws/mac/internal/tunnel"
)

const ifconfig = `lo0: flags=8049<UP,LOOPBACK,RUNNING,MULTICAST> mtu 16384
	inet 127.0.0.1 netmask 0xff000000
	inet6 ::1 prefixlen 128
en0: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	inet6 fe80::1%en0 prefixlen 64 scopeid 0x6
	inet 192.168.1.20 netmask 0xffffff00 broadcast 192.168.1.255
en5: flags=8822<BROADCAST,SMART,SIMPLEX,MULTICAST> mtu 1500
	inet 10.77.0.2 netmask 0xfffffffc broadcast 10.77.0.3
utun4: flags=8051<UP,POINTOPOINT,RUNNING,MULTICAST> mtu 1420
	inet 10.99.0.2 --> 10.99.0.2 netmask 0xffffffff
utun5: flags=8051<UP,POINTOPOINT,RUNNING,MULTICAST> mtu 1500
	inet 100.64.0.1 --> 100.64.0.1 netmask 0xffffffff
`

func TestParseIfconfig(t *testing.T) {
	addr := func(name, a string) tunnel.Iface { return tunnel.Iface{Name: name, Addr: netip.MustParseAddr(a)} }
	got := ParseIfconfig(ifconfig, "utun4")
	want := []tunnel.Iface{addr("en0", "192.168.1.20"), addr("lo0", "127.0.0.1"), addr("utun5", "100.64.0.1")}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestParseIfconfigEmpty(t *testing.T) {
	if got := ParseIfconfig("", "utun4"); len(got) != 0 {
		t.Errorf("got %v from no text", got)
	}
}
