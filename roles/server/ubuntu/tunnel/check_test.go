package tunnel

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/values"
)

func TestNetworkdCheck(t *testing.T) {
	for name, tc := range map[string]struct {
		active bool
		want   string
	}{
		"running":     {true, ""},
		"not running": {false, "systemd-networkd is not running"},
	} {
		t.Run(name, func(t *testing.T) {
			u := &effecttest.Units{States: map[string]effect.UnitState{"systemd-networkd.service": {Active: tc.active}}}
			err := networkdRuns(t.Context(), u)
			if (tc.want == "") != (err == nil) || (err != nil && !strings.Contains(err.Error(), tc.want)) {
				t.Errorf("networkdRuns = %v, want %q", err, tc.want)
			}
		})
	}
}

// networkdRunning is the units of a machine whose networkd runs.
func networkdRunning() *effecttest.Units {
	return &effecttest.Units{States: map[string]effect.UnitState{"systemd-networkd.service": {Active: true}}}
}

// twoPeers are the peers mac and tv.
var twoPeers = values.Tunnel{Peers: []values.TunnelPeer{{Name: "mac", Address: "10.99.0.2"}, {Name: "tv", Address: "10.99.0.3"}}}

func TestTunnelCheckNotesAMissingPeer(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Files["/etc/wireguard/mac.pub"] = []byte(keyA + "\n")
	if err := Check(t.Context(), m, networkdRunning(), out, twoPeers); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(out.Lines, "\n")
	if !strings.Contains(got, "leaves out the peer tv") || strings.Contains(got, "peer mac") {
		t.Errorf("said %q, want tv alone left out", got)
	}
	if !strings.Contains(got, "has no WireGuard key yet") {
		t.Errorf("said %q, want the key to come", got)
	}
}

func TestTunnelCheckRefusesABadPub(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/etc/wireguard/mac.pub"] = []byte("not a key\n")
	err := Check(t.Context(), m, networkdRunning(), &effecttest.Out{}, twoPeers)
	if err == nil || !strings.Contains(err.Error(), "/etc/wireguard/mac.pub holds no WireGuard public key") {
		t.Errorf("Check = %v, want the refusal", err)
	}
}

func TestTunnelCheckSaysThePublicKey(t *testing.T) {
	m, public := machine("")
	out := &effecttest.Out{}
	if err := Check(t.Context(), m, networkdRunning(), out, values.Tunnel{}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(out.Lines, "\n"); !strings.Contains(got, "for WS_PUBLIC_KEY in a server's values.yml: "+public) {
		t.Errorf("said %q, want the public key", got)
	}
}

func TestTunnelCheckRefusesABadKeyFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[keyFile] = []byte("x\n")
	if err := Check(t.Context(), m, networkdRunning(), &effecttest.Out{}, values.Tunnel{}); err == nil || !strings.Contains(err.Error(), "holds no WireGuard private key") {
		t.Errorf("Check = %v, want the refusal", err)
	}
}
