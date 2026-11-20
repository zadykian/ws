package cli

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/resource/ufw"
)

const (
	ikewardSysctl = "/etc/sysctl.d/99-ikeward.conf"
	// ikewardPeer gives ikeward's values one peer, 10.99.0.2.
	ikewardPeer = ikewardValues + "WS_IKEWARD_FORWARD_FROM: [10.99.0.2]\n"
)

// ikewardUnits are ikeward's seven units.
var ikewardUnits = []string{"guard/ikeward", "binary/ikeward", "file/ikeward-sysctl", "state/ikeward-forward",
	"state/ikeward-install", "state/ikeward-status", "state/ikeward-sysctl"}

func TestIkewardSpecOffWithoutProfile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	units, out := cableUnits(t, m, "---\nWS_IKEWARD_FORWARD_FROM: [10.99.0.2]\n")
	for _, name := range ikewardUnits {
		if got := applyUnit(t, units, name); got != nil {
			t.Errorf("%s planned %q, want nothing", name, got)
		}
		if want := "say: " + name + ": off on this machine, as its when gives no set"; !slices.Contains(out.Lines, want) {
			t.Errorf("said %q, want %q", out.Lines, want)
		}
	}
	if len(m.Calls) != 0 {
		t.Errorf("calls %q, want none", m.Calls)
	}
}

func TestIkewardSysctlAbsentWithoutPeers(t *testing.T) {
	want, err := os.ReadFile("../ubuntu/ikeward/99-ikeward.conf")
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	m.Dirs["/etc"], m.Dirs["/etc/sysctl.d"] = 0o755, 0o755
	units, _ := cableUnits(t, m, ikewardPeer)
	applyUnit(t, units, "file/ikeward-sysctl")
	if got := m.Files[ikewardSysctl]; string(got) != string(want) || m.Modes[ikewardSysctl] != 0o644 {
		t.Errorf("the file %q at %o, want 99-ikeward.conf at 644", got, m.Modes[ikewardSysctl])
	}
	units, out := cableUnits(t, m, ikewardValues)
	if got := applyUnit(t, units, "file/ikeward-sysctl"); len(got) != 1 {
		t.Errorf("plan %q, want the file removed", got)
	}
	if !slices.Contains(m.Calls, "root rm -rf -- "+ikewardSysctl) {
		t.Errorf("calls %q, want the file removed", m.Calls)
	}
	if want := "say: state/ikeward-sysctl: off on this machine, as its when gives no set"; applyUnit(t, units, "state/ikeward-sysctl") != nil ||
		!slices.Contains(out.Lines, want) {
		t.Errorf("said %q, want %q", out.Lines, want)
	}
}

func TestIkewardForwardRules(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	// The peer 10.99.0.9 is no longer listed; ikeward is 696b6577617264 in hex.
	m.Files[ufw.UserRules] = []byte("### tuple ### route:allow any any 0.0.0.0/0 any 10.99.0.9 in_wg0!out_ikeward0 comment=696b6577617264\n")
	units, _ := cableUnits(t, m, ikewardPeer)
	steps := map[string]kind.Unit{}
	for _, u := range units {
		steps[u.Name] = u
	}
	p, err := steps["state/ikeward-forward"].Step.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range p.Changes {
		got = append(got, c.(interface{ String() string }).String())
	}
	want := []string{"ufw route allow in on wg0 out on ikeward0 from 10.99.0.2 to any",
		"ufw route delete allow in on wg0 out on ikeward0 from 10.99.0.9 to any"}
	if !slices.Equal(got, want) {
		t.Errorf("plan %q, want %q", got, want)
	}
	m.Results["ufw --force route delete allow in on wg0 out on ikeward0 from 10.99.0.9 to any comment ikeward"] = effecttest.Result{}
	units, _ = cableUnits(t, m, ikewardValues)
	if got := applyUnit(t, units, "state/ikeward-forward"); len(got) != 1 || !strings.Contains(got[0], "delete") {
		t.Errorf("no peer: plan %q, want the rule deleted", got)
	}
}

func TestIkewardBinaryPins(t *testing.T) {
	for arch, sum := range map[string]string{
		"amd64": "b78ba9456f246ddaa5f08f34706cf96b975ad3717a69f142cbcbd893294b2d23",
		"arm64": "711b9a1c72b59ce7bc453d94c790855588b529803b8789abc72ed3b405e83e3b",
	} {
		url := "https://github.com/zadykian/ikeward/releases/download/v0.3.0/ikeward-linux-" + arch
		m := effecttest.NewMachine()
		m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte(arch + "\n")}
		m.Pages[url] = []byte("not ikeward")
		units, _ := cableUnits(t, m, ikewardValues)
		i := slices.IndexFunc(units, func(u kind.Unit) bool { return u.Name == "binary/ikeward" })
		p, err := units[i].Step.Plan(t.Context())
		if err == nil {
			err = units[i].Step.Apply(t.Context(), p)
		}
		if err == nil || !strings.Contains(err.Error(), "want "+sum) || !slices.Contains(m.Calls, "download "+url) {
			t.Errorf("%s: apply = %v, calls %q; want %s refused by its pin", arch, err, m.Calls, url)
		}
		if _, ok := m.Files["/usr/local/sbin/ikeward"]; ok {
			t.Errorf("%s: the binary was written", arch)
		}
	}
}

func TestIkewardGuardRunsTheCheck(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[selfPath+" server ubuntu ikeward check"] = effecttest.Result{Err: errors.New("WS_IKEWARD_P12 or WS_IKEWARD_SSWAN is empty")}
	units, _ := cableUnits(t, m, ikewardValues)
	i := slices.IndexFunc(units, func(u kind.Unit) bool { return u.Name == "guard/ikeward" })
	if _, err := units[i].Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "WS_IKEWARD_P12 or WS_IKEWARD_SSWAN is empty") {
		t.Errorf("plan = %v, want the check's failure", err)
	}
}

func TestIkewardOrder(t *testing.T) {
	units, _ := cableUnits(t, effecttest.NewMachine(), ikewardPeer)
	names := kind.Names(units)
	at := func(name string) int { return slices.Index(names, name) }
	for _, pair := range [][2]string{
		{"guard/ikeward", "binary/ikeward"},
		{"guard/ikeward", "file/ikeward-sysctl"},
		{"binary/ikeward", "state/ikeward-install"},
		{"state/ikeward-install", "state/ikeward-status"},
		{"file/ikeward-sysctl", "state/ikeward-sysctl"},
		{"state/ikeward-sysctl", "state/ikeward-forward"},
	} {
		if at(pair[0]) < 0 || at(pair[0]) > at(pair[1]) {
			t.Errorf("%s runs at %d, after %s at %d", pair[0], at(pair[0]), pair[1], at(pair[1]))
		}
	}
}
