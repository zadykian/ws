package otelcol_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/ubuntu/values"
)

const (
	dockerSocket  = "/etc/systemd/system/otelcol-docker.socket"
	dockerService = "/etc/systemd/system/otelcol-docker.service"
)

// Docker's containers reach the collector through a socket on the bridge address, which passes
// each connection to the receiver on loopback.
func TestDockerSocketProxiesTheBridgeToTheReceiver(t *testing.T) {
	m, e := machine(t, "")
	v, err := values.Parse([]byte("WS_DOCKER_BRIDGE_IP: 172.18.0.1\n"))
	if err != nil {
		t.Fatal(err)
	}
	e.Data = v
	if _, err := step.Run(t.Context(), kind.Steps(only(loaded(t, e), "service/otelcol", "service/otelcol-docker")), e.Out, false); err != nil {
		t.Fatal(err)
	}
	for path, wants := range map[string][]string{
		dockerSocket:  {"ListenStream=172.18.0.1:4317\n", "FreeBind=yes\n", "WantedBy=sockets.target\n"},
		dockerService: {"ExecStart=/usr/lib/systemd/systemd-socket-proxyd 127.0.0.1:4317\n", "Requires=otelcol-docker.socket\n"},
	} {
		for _, want := range wants {
			if !strings.Contains(string(m.Files[path]), want) {
				t.Errorf("%s lacks %q:\n%s", path, want, m.Files[path])
			}
		}
	}
	u := e.U.(*effecttest.Units)
	for _, want := range []string{"enable otelcol-docker.socket", "start otelcol-docker.socket"} {
		if !strings.Contains(strings.Join(u.Calls, "\n"), want) {
			t.Errorf("calls %q lack %q", u.Calls, want)
		}
	}
}

// A second run changes nothing, and a moved bridge address restarts the socket.
func TestDockerSocketFollowsTheBridgeAddress(t *testing.T) {
	m, e := machine(t, "")
	units := func() []kind.Unit { return only(loaded(t, e), "service/otelcol", "service/otelcol-docker") }
	if _, err := step.Run(t.Context(), kind.Steps(units()), e.Out, false); err != nil {
		t.Fatal(err)
	}
	if p, err := units()[1].Step.Plan(t.Context()); err != nil || len(p.Changes) != 0 {
		t.Fatalf("a second plan has %v, %v; want none", p.Changes, err)
	}
	v, err := values.Parse([]byte("WS_DOCKER_BRIDGE_IP: 172.19.0.1\n"))
	if err != nil {
		t.Fatal(err)
	}
	e.Data = v
	u := e.U.(*effecttest.Units)
	u.Calls = nil
	if _, err := step.Run(t.Context(), kind.Steps(units()), e.Out, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(m.Files[dockerSocket]), "ListenStream=172.19.0.1:4317\n") {
		t.Errorf("%s =\n%s\nwant the new address", dockerSocket, m.Files[dockerSocket])
	}
	if !slices.Contains(u.Calls, "restart otelcol-docker.socket") {
		t.Errorf("calls %q lack the socket's restart", u.Calls)
	}
}
