package setup_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/setup"
)

func tunnelStep(m *effecttest.Machine) *setup.Tunnel {
	return &setup.Tunnel{Home: "/h", M: m, Out: &effecttest.Out{}, Self: func() ([]byte, error) { return []byte("ws"), nil }}
}

func lines(p []string) string { return strings.Join(p, "\n") }

func tunnelChanges(t *testing.T, s *setup.Tunnel) []string {
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range p.Changes {
		got = append(got, c.String())
	}
	return got
}

func TestTunnelPlansTheCopyAndTheKeyOnAFreshMac(t *testing.T) {
	got := lines(tunnelChanges(t, tunnelStep(effecttest.NewMachine())))
	want := lines([]string{
		"would build ws into /h/.local/bin/ws",
		"would make /usr/local, root's, with mode 755",
		"would make /usr/local/libexec, root's, with mode 755",
		"would make /usr/local/libexec/ws-tunnel, root's, with mode 755",
		"would copy /h/.local/bin/ws to /usr/local/libexec/ws-tunnel/ws",
		"create directory /etc/ws-tunnel, owner 0:0, mode 700",
		"create directory /etc/ws-tunnel/routes.d, owner 0:0, mode 755",
		"would make the Mac's WireGuard key, /etc/ws-tunnel/private.key",
	})
	if got != want {
		t.Errorf("Plan =\n%s\nwant\n%s", got, want)
	}
}

func TestTunnelPlansNothingOnceDone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/.local/bin/ws"] = []byte("ws")
	for path, mode := range map[string]effect.Attrs{
		"/usr/local": {Mode: 0o755}, "/usr/local/libexec": {Mode: 0o755}, "/usr/local/libexec/ws-tunnel": {Mode: 0o755},
		"/etc/ws-tunnel": {Mode: 0o700}, "/etc/ws-tunnel/routes.d": {Mode: 0o755}, "/etc/ws-tunnel/private.key": {Mode: 0o600},
	} {
		m.Root[path] = mode
	}
	m.Results["/usr/local/libexec/ws-tunnel/ws mac tunnel key"] = effecttest.Result{Out: []byte("PUB\n")}
	m.Files["/h/.config/ws/tunnel.pub"] = []byte("PUB\n")
	m.Dirs["/h/.config"], m.Dirs["/h/.config/ws"] = 0o755, 0o700
	m.Files["/usr/local/libexec/ws-tunnel/ws"] = []byte("ws")
	m.Root["/usr/local/libexec/ws-tunnel/ws"] = effect.Attrs{Mode: 0o755}
	if got := tunnelChanges(t, tunnelStep(m)); len(got) != 0 {
		t.Errorf("Plan = %q; want none", got)
	}
}

func TestTunnelStopsWhereADirectoryAboveTheCopyIsWritable(t *testing.T) {
	m := effecttest.NewMachine()
	m.Root["/usr/local"] = effect.Attrs{UID: 501, GID: 20, Mode: 0o755}
	_, err := tunnelStep(m).Plan(context.Background())
	if err == nil || err.Error() != "/usr/local, above root's copy, is not root's alone: owner, group and mode 501:20 755" {
		t.Errorf("Plan error = %v", err)
	}
}

// TestTunnelApplyInstallsCopiesAndMakesTheKey applies a fresh Mac's plan, then plans again. With the
// key that root's ws makes, and ws where mv put it, that plan is empty.
func TestTunnelApplyInstallsCopiesAndMakesTheKey(t *testing.T) {
	m := effecttest.NewMachine()
	s := tunnelStep(m)
	for _, line := range []string{"mkdir -p /h/.local/bin", "mv -f /h/.local/bin/ws.new /h/.local/bin/ws",
		"chown 0:0 /usr/local", "chown 0:0 /usr/local/libexec", "chown 0:0 /usr/local/libexec/ws-tunnel"} {
		m.Results[line] = effecttest.Result{}
	}
	m.Results["/usr/local/libexec/ws-tunnel/ws mac tunnel key"] = effecttest.Result{Out: []byte("PUB\n")}
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if string(m.Files["/usr/local/libexec/ws-tunnel/ws"]) != "ws" || m.Modes["/usr/local/libexec/ws-tunnel/ws"] != 0o755 {
		t.Errorf("root's copy holds %q", m.Files["/usr/local/libexec/ws-tunnel/ws"])
	}
	out := lines(s.Out.(*effecttest.Out).Lines)
	if !strings.Contains(out, "say: copied /h/.local/bin/ws to /usr/local/libexec/ws-tunnel/ws\n") ||
		!strings.Contains(out, "say: made the Mac's WireGuard key, /etc/ws-tunnel/private.key") {
		t.Errorf("out = %q; want the copy and the key", out)
	}
	m.Files["/h/.local/bin/ws"] = []byte("ws")
	m.Root["/etc/ws-tunnel/private.key"] = effect.Attrs{Mode: 0o600}
	if got := tunnelChanges(t, tunnelStep(m)); len(got) != 0 {
		t.Errorf("a second Plan = %q; want none", got)
	}
}
