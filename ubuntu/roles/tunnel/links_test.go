package tunnel

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/internal/slot"
	"github.com/zadykian/ws/internal/wgkey"
	"github.com/zadykian/ws/ubuntu/values"
)

// labTunnel is the tunnel of the guest lab, at slot 1 and port 51821, with the peer mac.
func labTunnel() values.GuestTunnel {
	return values.GuestTunnel{
		Guest: values.Guest{Name: "lab"}, Slot: slot.Slot(1), Port: 51821,
		Peers: []values.TunnelPeer{{Name: "mac", Address: "10.99.1.2"}, {Name: "tv", Address: "10.99.1.3"}},
	}
}

// linksMachine holds the group systemd-network, lab's key and mac's key file, where tv has none.
func linksMachine() (*effecttest.Machine, string) {
	m := effecttest.NewMachine()
	m.Files["/etc/group"] = []byte("root:x:0:\nsystemd-network:x:998:\n")
	private, _ := wgkey.Generate(bytes.NewReader(bytes.Repeat([]byte{5}, 32)))
	public, _ := wgkey.Public(private)
	m.Files[labTunnel().KeyFile()] = []byte(private + "\n")
	m.Files["/etc/wireguard/wg1/mac.pub"] = []byte(keyA + "\n")
	m.Dirs["/etc/systemd"], m.Dirs["/etc/systemd/network"] = 0o755, 0o755
	return m, public
}

// firstLines cuts each headline to its first line, where a file's diff follows.
func firstLines(heads []string) []string {
	var got []string
	for _, h := range heads {
		first, _, _ := strings.Cut(h, "\n")
		got = append(got, first)
	}
	return got
}

func newLinks(m *effecttest.Machine) Links {
	return Links{M: m, Out: &effecttest.Out{}, Tunnels: []values.GuestTunnel{labTunnel()}, Tries: 2, Sleep: func(time.Duration) {}}
}

func TestANewGuestTunnelGetsItsFilesAReloadAndAWait(t *testing.T) {
	m, _ := linksMachine()
	got := firstLines(headlines(t, newLinks(m)))
	if len(got) != 4 || !strings.HasPrefix(got[0], "create /etc/systemd/network/60-wg1.netdev") ||
		!strings.HasPrefix(got[1], "create /etc/systemd/network/60-wg1.network") ||
		!strings.HasPrefix(got[2], "reload networkd for the guests' tunnels: wg1's files change") || got[3] != "wait for wg1 to hold the server's key" {
		t.Errorf("plan = %q", got)
	}
}

func TestAGuestTunnelsFilesNameItsSlotItsPortAndItsPeersWithAKey(t *testing.T) {
	m, _ := linksMachine()
	cs, err := newLinks(m).Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cs[:2] {
		if err := c.Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	netdev := string(m.Files["/etc/systemd/network/60-wg1.netdev"])
	for _, want := range []string{"Name=wg1\n", "PrivateKeyFile=/etc/systemd/network/60-wg1.key\n", "ListenPort=51821\n",
		"PublicKey=" + keyA + "\nAllowedIPs=10.99.1.2/32"} {
		if !strings.Contains(netdev, want) {
			t.Errorf(".netdev lacks %q:\n%s", want, netdev)
		}
	}
	if strings.Contains(netdev, "10.99.1.3") {
		t.Errorf(".netdev holds tv, whose key file is missing:\n%s", netdev)
	}
	network := string(m.Files["/etc/systemd/network/60-wg1.network"])
	if !strings.Contains(network, "Name=wg1\n") || !strings.Contains(network, "Destination=10.99.1.0/24\n") || strings.Contains(network, "Address=") {
		t.Errorf(".network:\n%s", network)
	}
	if mode, owner := m.Modes["/etc/systemd/network/60-wg1.netdev"], m.Owners["/etc/systemd/network/60-wg1.netdev"]; mode != 0o640 || owner != [2]int{0, 998} {
		t.Errorf(".netdev mode %o, owner %v, want root:998 at 640", mode, owner)
	}
}

func TestAGuestTunnelAsWantedPlansNothing(t *testing.T) {
	m, public := linksMachine()
	cs, _ := newLinks(m).Plan(t.Context())
	for _, c := range cs[:2] {
		if err := c.Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	m.Dirs["/sys/class/net/wg1"] = 0o755
	m.Results["wg show wg1 public-key"] = effecttest.Result{Out: []byte(public + "\n")}
	if got := headlines(t, newLinks(m)); len(got) != 0 {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestAnInterfaceOfAnotherKeyIsDeletedBeforeTheReload(t *testing.T) {
	m, _ := linksMachine()
	cs, _ := newLinks(m).Plan(t.Context())
	for _, c := range cs[:2] {
		if err := c.Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	m.Dirs["/sys/class/net/wg1"] = 0o755
	m.Results["wg show wg1 public-key"] = effecttest.Result{Out: []byte(keyA + "\n")}
	got := headlines(t, newLinks(m))
	if len(got) != 3 || got[0] != "delete wg1, so that networkd creates it with the key" || !strings.Contains(got[1], "wg1 holds another key") {
		t.Errorf("plan = %q, want the deletion, the reload and the wait", got)
	}
}

func TestAMissingKeyIsAnErrorNotAPlan(t *testing.T) {
	m, _ := linksMachine()
	delete(m.Files, labTunnel().KeyFile())
	if _, err := newLinks(m).Plan(t.Context()); err == nil || !slices.Contains(strings.Fields(err.Error()), "missing:") {
		t.Errorf("err = %v, want the key reported missing", err)
	}
}

func TestGuestKeysAreMadeAtTheirOwnFiles(t *testing.T) {
	m, _ := linksMachine()
	delete(m.Files, labTunnel().KeyFile())
	keys := GuestKeys(m, &effecttest.Out{}, []values.GuestTunnel{labTunnel()})
	var got []string
	for _, k := range keys {
		got = append(got, headlines(t, k)...)
	}
	if len(got) != 1 || !strings.Contains(got[0], "make the server's private key") {
		t.Errorf("plan = %q", got)
	}
	var _ resource.Resource = Links{}
}
