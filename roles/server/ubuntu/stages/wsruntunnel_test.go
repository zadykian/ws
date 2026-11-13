package stages

import (
	"context"
	"encoding/base64"
	"errors"
	"net/netip"
	"slices"
	"strings"
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

// tunnelRun is newRun over a Want at the address and the port, whose host_vars reads as file.
func tunnelRun(address, port, file string) (*WsRun, *effecttest.Machine) {
	r, m := newRun("lan", idleRun, false, false)
	var a netip.Prefix
	if address != "" {
		a = netip.MustParsePrefix(address)
	}
	r.Ws = session.New(wantReach{fileReach{sessiontest.Reach{M: m}}, a, port},
		session.Scripts{Remote: []byte("x"), Files: []byte("y")})
	m.Results[sessiontest.Op("status")] = effecttest.Result{Out: sessiontest.Text(idleRun)}
	m.Results[sessiontest.Op("values")] = effecttest.Result{Out: []byte(file)}
	return r, m
}

// wantReach is a Reach whose Want names a tunnel, as the Mac's conf does at a slot.
type wantReach struct {
	fileReach
	address netip.Prefix
	port    string
}

func (r wantReach) Want() (session.Want, error) {
	w, err := r.fileReach.Want()
	w.Address, w.Port = r.address, r.port
	return w, err
}

func TestWsRunPlansWhatHostVarsLacks(t *testing.T) {
	for name, tc := range map[string]struct{ address, port, file, want string }{
		"both":    {"10.99.1.1/24", "51821", "", "after naming wireguard_address: 10.99.1.1/24 and wireguard_port: 51821 in ws's host_vars"},
		"address": {"10.99.1.1/24", "", "", "after naming wireguard_address: 10.99.1.1/24 in ws's host_vars"},
		"held":    {"10.99.1.1/24", "51821", "wireguard_address: 10.99.1.1/24\nwireguard_port: 51821\n", ""},
		"slot 0":  {"10.99.0.1/24", "", "", ""},
	} {
		r, m := tunnelRun(tc.address, tc.port, "---\n"+tc.file)
		p, err := r.Plan(context.Background())
		got := ""
		if len(p.Changes) > 0 {
			got = p.Changes[0].String()
		}
		if err != nil || (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("%s: Plan = %q, %v; want %q", name, got, err, tc.want)
		}
		if tc.address == "10.99.0.1/24" && slices.Contains(m.Calls, "run op values") {
			t.Errorf("%s: calls = %q; want no values op", name, m.Calls)
		}
	}
}

func TestWsRunStopsWhereHostVarsCannotBeRead(t *testing.T) {
	r, m := tunnelRun("10.99.1.1/24", "", "")
	m.Results[sessiontest.Op("values")] = effecttest.Result{Err: errors.New("ssh: no route")}
	if _, err := r.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "host_vars on ws") {
		t.Errorf("Plan where the read fails = %v; want the stop", err)
	}
}

// pairsOp is the op that writes the pairs, each sent as base64, as the files script reads it.
func pairsOp(pairs ...string) string {
	words := []string{"values"}
	for _, p := range pairs {
		words = append(words, base64.StdEncoding.EncodeToString([]byte(p)))
	}
	return sessiontest.Op(words...)
}

func TestWsRunWritesTheTunnelBeforeTheUser(t *testing.T) {
	r, m := tunnelRun("10.99.1.1/24", "51821", "---\n")
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\nWS_USER=alice\n")
	p, err := r.Plan(context.Background())
	if err != nil || len(p.Changes) != 1 {
		t.Fatalf("Plan = %v, %v; want one change", p.Changes, err)
	}
	pairs := pairsOp("wireguard_address=10.99.1.1/24", "wireguard_port=51821")
	m.Results[pairs] = effecttest.Result{}
	m.Results[sessiontest.Op("user", "alice")] = effecttest.Result{}
	m.Results[sessiontest.Op("run")] = effecttest.Result{}
	if err := r.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	at := func(op string) int { return slices.Index(m.Calls, "run "+op) }
	write, user, run := at(pairs), at("op user alice"), at("op run")
	if write < 0 || write > user || user > run {
		t.Errorf("calls = %q; want the pairs, the user, then the run", m.Calls)
	}
}

func TestWsRunStopsWhereTheTunnelCannotBeWritten(t *testing.T) {
	r, m := tunnelRun("10.99.1.1/24", "", "---\n")
	p, err := r.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	m.Results[pairsOp("wireguard_address=10.99.1.1/24")] = effecttest.Result{Err: errors.New("denied")}
	err = r.Apply(context.Background(), p)
	if err == nil || !strings.Contains(err.Error(), "cannot write wireguard_address: 10.99.1.1/24 into ws's host_vars") {
		t.Errorf("Apply where the write fails = %v; want the stop", err)
	}
	if slices.Contains(m.Calls, "run op run") {
		t.Errorf("calls = %q; want no run", m.Calls)
	}
}
