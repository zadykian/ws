package wireguard

import (
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestWg0AppliesTheCommands(t *testing.T) {
	m, _ := machine("OTHER")
	m.Results["ip link delete wg0"] = effecttest.Result{}
	m.Results["networkctl reload"] = effecttest.Result{}
	cs, _ := wg0{m: m}.Plan(t.Context())
	for _, c := range cs {
		if err := c.Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	if got := strings.Join(m.Calls[len(m.Calls)-2:], ";"); got != "root ip link delete wg0;root networkctl reload" {
		t.Errorf("calls end %q", got)
	}
}

func TestWg0SkipsWithoutSystemd(t *testing.T) {
	m, _ := machine("OTHER")
	delete(m.Dirs, "/run/systemd/system")
	if got := descriptions(t, wg0{m: m}); len(got) != 0 {
		t.Errorf("plan = %q, want nothing", got)
	}
}

func TestWg0ReportsWhereWgShowFails(t *testing.T) {
	m, _ := machine("OTHER")
	m.Results["wg show wg0 public-key"] = effecttest.Result{Err: errors.New("no device")}
	if _, err := (wg0{m: m}).Plan(t.Context()); err == nil {
		t.Error("Plan = nil, want the error")
	}
}
