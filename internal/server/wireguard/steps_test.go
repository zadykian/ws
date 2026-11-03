package wireguard

import (
	"bytes"
	"encoding/hex"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/internal/wgkey"
	"github.com/zadykian/ws/step"
)

const peerKey = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB="

// fakeSteps returns steps on a machine that systemd booted, with the packages installed, the
// peer mac's key in place and the firewall rule let in.
func fakeSteps() (*effecttest.Machine, []step.Step) {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	m.Files["/etc/group"] = []byte("systemd-network:x:998:\n")
	m.Files["/etc/wireguard/mac.pub"] = []byte(peerKey + "\n")
	m.Files[ufwRules] = []byte("### tuple ### allow udp 51820 0.0.0.0/0 any 0.0.0.0/0 in comment=" +
		hex.EncodeToString([]byte("WireGuard")) + "\n")
	m.Results["dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n wireguard-tools systemd iptables"] =
		effecttest.Result{Out: []byte("wireguard-tools ii\nsystemd ii\niptables ii\n")}
	m.Results["networkctl reload"] = effecttest.Result{}
	u := &effecttest.Units{States: map[string]effect.UnitState{"systemd-networkd.service": {Active: true}}}
	v := values.Default()
	random := bytes.NewReader(bytes.Repeat([]byte{5}, 32))
	return m, steps(v, m, u, &effecttest.Out{}, random, func(time.Duration) {})
}

const ufwRules = "/etc/ufw/user.rules"

func TestStepsAreInOrderAndWaitOnTheirPredecessors(t *testing.T) {
	_, ss := fakeSteps()
	var names, after []string
	for _, s := range ss {
		names, after = append(names, s.Name()), append(after, s.WaitsOn())
	}
	wantNames := []string{settingsStep, packagesStep, dropsStep, tunnelStep, keyStep}
	wantAfter := []string{"", settingsStep, packagesStep, dropsStep, tunnelStep}
	if !slices.Equal(names, wantNames) || !slices.Equal(after, wantAfter) {
		t.Errorf("steps = %q after %q, want %q after %q", names, after, wantNames, wantAfter)
	}
}

func TestStepsMakeTheKeyAndTheFilesOnceAndThenPlanNothing(t *testing.T) {
	m, ss := fakeSteps()
	m.Dirs["/sys/class/net/wg0"] = 0o755
	// wg0 holds the key that the first run makes, from the five bytes the fake reads.
	private, _ := wgkey.Generate(bytes.NewReader(bytes.Repeat([]byte{5}, 32)))
	public, _ := wgkey.Public(private)
	m.Results["wg show wg0 public-key"] = effecttest.Result{Out: []byte(public + "\n")}
	m.Results["ip link delete wg0"] = effecttest.Result{}
	out := &effecttest.Out{}
	if changed, err := step.Run(t.Context(), ss, out, false); err != nil || !changed {
		t.Fatalf("first run = %v, %v; want changes", changed, err)
	}
	for path, want := range map[string]string{
		keyFile:     private,
		netdevFile:  "PublicKey=" + peerKey,
		networkFile: "Address=10.99.0.1/24",
		dropsScript: "WS-MAC",
		dropsUnit:   "ExecStart=" + dropsScript + " wg0\n",
	} {
		if !strings.Contains(string(m.Files[path]), want) {
			t.Errorf("%s = %q, want %q in it", path, m.Files[path], want)
		}
	}
	if a, _ := m.RootAttrs(keyFile); a != (effect.Attrs{GID: 998, Mode: 0o640}) {
		t.Errorf("key attrs = %+v", a)
	}
	if changed, err := step.Run(t.Context(), ss, out, false); err != nil || changed {
		t.Errorf("second run = %v, %v; want nothing to change", changed, err)
	}
}
