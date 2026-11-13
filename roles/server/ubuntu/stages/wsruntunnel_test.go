package stages

import (
	"context"
	"net/netip"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
)

// pairsFor runs tunnelPairs for the address and port over a host_vars that reads as file, and
// returns the pairs and the ops that ran.
func pairsFor(address, port, file string) ([]string, []string, error) {
	m := effecttest.NewMachine()
	m.Results[sessiontest.Op("values")] = effecttest.Result{Out: []byte(file)}
	var w session.Want
	if address != "" {
		w.Address = netip.MustParsePrefix(address)
	}
	w.Port = port
	pairs, err := tunnelPairs(context.Background(), sessiontest.On(m, idleRun), w)
	return pairs, m.Calls, err
}

func TestTunnelPairsReadsNothingAtTheDefaults(t *testing.T) {
	for name, tc := range map[string]struct{ address, port string }{
		"slot 0":    {"10.99.0.1/24", ""},
		"the port":  {"10.99.0.1/24", "51820"},
		"port only": {"", "51820"},
	} {
		pairs, calls, err := pairsFor(tc.address, tc.port, "")
		if err != nil || len(pairs) != 0 || len(calls) != 0 {
			t.Errorf("%s: pairs = %q, calls = %q, %v; want none", name, pairs, calls, err)
		}
	}
}

func TestTunnelPairsKeepsWhatHostVarsLacks(t *testing.T) {
	both := []string{"wireguard_address=10.99.1.1/24", "wireguard_port=51821"}
	for name, tc := range map[string]struct {
		address, port, file string
		want                []string
	}{
		"both":         {"10.99.1.1/24", "51821", "", both},
		"address":      {"10.99.1.1/24", "", "", both[:1]},
		"port":         {"", "51821", "", both[1:]},
		"port is held": {"10.99.1.1/24", "51821", "wireguard_port: 51821\n", both[:1]},
		"both held":    {"10.99.1.1/24", "51821", "wireguard_address: 10.99.1.1/24\nwireguard_port: 51821\n", nil},
	} {
		pairs, _, err := pairsFor(tc.address, tc.port, "---\n"+tc.file)
		if err != nil || !slices.Equal(pairs, tc.want) {
			t.Errorf("%s: pairs = %q, %v; want %q", name, pairs, err, tc.want)
		}
	}
}
