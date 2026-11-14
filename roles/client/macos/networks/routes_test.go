package networks_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/networks"
)

func TestRoutesPutsTheNetworksInTheFileOfTheirSet(t *testing.T) {
	m := effecttest.NewMachine()
	r := &networks.Routes{Set: "lab", Networks: []string{"10.1.0.0/24"}, M: m}
	if _, err := r.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "routes.d is missing") {
		t.Errorf("Plan error = %v", err)
	}
	m.Root["/etc/ws-tunnel/routes.d"] = effect.Attrs{Mode: 0o755}
	p, err := r.Plan(context.Background())
	if err != nil || len(p.Changes) != 1 || !strings.Contains(p.Changes[0].String(), "routes.d/lab") {
		t.Errorf("Plan = %v, %v", p, err)
	}
}

func TestRoutesRefusesWhatTheShellRefused(t *testing.T) {
	for set, cidr := range map[string]string{"Lab": "10.1.0.0/24", "lab": "10.1.0.5/24", "a_b": "10.99.0.0/16",
		"x": "10.77.0.0/29", "y": "0.0.0.0/0", "z": "10.1.0.0"} {
		r := &networks.Routes{Set: set, Networks: []string{cidr}, M: effecttest.NewMachine()}
		if _, err := r.Plan(context.Background()); err == nil {
			t.Errorf("Plan of %q %q succeeded", set, cidr)
		}
	}
}

func TestRoutesAloneRefusesAMissingRoutesDirUnderCheck(t *testing.T) {
	r := &networks.Routes{Set: "lab", Networks: []string{"10.1.0.0/24"}, M: effecttest.NewMachine(), Check: true, Alone: true}
	if _, err := r.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "routes.d is missing") {
		t.Errorf("Plan error = %v", err)
	}
}

func TestRoutesSignalsEachLoadedDaemonAlone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Root["/etc/ws-tunnel/routes.d"] = effect.Attrs{Mode: 0o755}
	installDaemons(m, "lab", "ws")
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{}
	p, err := (&networks.Routes{Set: "lab", Networks: []string{"10.1.0.0/24"}, M: m}).Plan(context.Background())
	if err != nil || len(p.Changes) != 2 || p.Changes[1].String() != "would run launchctl kill SIGHUP system/com.github.zadykian.ws-tunnel.ws" {
		t.Errorf("Plan = %v, %v; want the file and a signal to ws alone", p, err)
	}
}
