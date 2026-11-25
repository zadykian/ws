package tunnel

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/ubuntu/values"
)

// removeMachine holds lab's tunnel at slot 1, its interface, and the host's own wg0 with its peer.
func removeMachine() *effecttest.Machine {
	m, _ := linksMachine()
	for _, f := range []string{"/etc/systemd/network/60-wg1.netdev", "/etc/systemd/network/60-wg1.network",
		"/etc/systemd/network/60-wg0.key", "/etc/systemd/network/60-wg0.netdev", "/etc/wireguard/mac.pub"} {
		m.Files[f] = []byte("x\n")
	}
	m.Dirs["/etc/wireguard"], m.Dirs["/etc/wireguard/wg1"], m.Dirs["/sys/class/net/wg1"] = 0o700, 0o700, 0o755
	m.Results["ip link delete wg1"] = effecttest.Result{}
	return m
}

func TestRemoveGuestDeletesTheSlotsKeyPeersFilesAndInterfaceAlone(t *testing.T) {
	m := removeMachine()
	if err := RemoveGuest(t.Context(), m, labTunnel()); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{labTunnel().KeyFile(), "/etc/wireguard/wg1/mac.pub",
		"/etc/systemd/network/60-wg1.netdev", "/etc/systemd/network/60-wg1.network"} {
		if _, ok := m.Files[f]; ok {
			t.Errorf("%s stays", f)
		}
	}
	if _, ok := m.Dirs["/etc/wireguard/wg1"]; ok {
		t.Errorf("the peers' directory stays")
	}
	for _, f := range []string{"/etc/systemd/network/60-wg0.key", "/etc/systemd/network/60-wg0.netdev", "/etc/wireguard/mac.pub"} {
		if _, ok := m.Files[f]; !ok {
			t.Errorf("%s, wg0's, is gone", f)
		}
	}
	del := slices.Index(m.Calls, "root ip link delete wg1")
	if del < 0 || del < slices.Index(m.Calls, "root rm -rf -- /etc/wireguard/wg1") {
		t.Errorf("calls %q do not delete wg1 after its files", m.Calls)
	}
}

func TestRemoveGuestLeavesAMissingInterfaceAndRefusesSlotZero(t *testing.T) {
	m := removeMachine()
	delete(m.Dirs, "/sys/class/net/wg1")
	if err := RemoveGuest(t.Context(), m, labTunnel()); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(m.Calls, "root ip link delete wg1") {
		t.Errorf("a missing interface is deleted: %q", m.Calls)
	}
	m = removeMachine()
	if err := RemoveGuest(t.Context(), m, values.GuestTunnel{Guest: values.Guest{Name: "lab"}}); err == nil {
		t.Errorf("slot 0, wg0's, is removed")
	}
	if len(m.Calls) != 0 {
		t.Errorf("slot 0: calls %q", m.Calls)
	}
}
