package wireguard

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/server/values"
)

func TestSettingsCheckIsAPlanError(t *testing.T) {
	if _, err := (settingsCheck{values.Wireguard{Address: "x", Port: 1}}).Plan(t.Context()); err == nil {
		t.Error("Plan = nil, want the refusal")
	}
	w := values.Wireguard{Address: "10.99.0.1/24", Port: 51820}
	if cs, err := (settingsCheck{w}).Plan(t.Context()); err != nil || len(cs) != 0 {
		t.Errorf("Plan = %v, %v; want nothing", cs, err)
	}
}

func TestNetworkdCheck(t *testing.T) {
	for name, tc := range map[string]struct {
		booted, active bool
		want           string
	}{
		"running":     {true, true, ""},
		"not running": {true, false, "systemd-networkd is not running"},
		"a container": {false, false, ""},
	} {
		t.Run(name, func(t *testing.T) {
			m, u := effecttest.NewMachine(), &effecttest.Units{States: map[string]effect.UnitState{"systemd-networkd.service": {Active: tc.active}}}
			if tc.booted {
				m.Dirs["/run/systemd/system"] = 0o755
			}
			_, err := (networkdCheck{m, u}).Plan(t.Context())
			if (tc.want == "") != (err == nil) || (err != nil && !strings.Contains(err.Error(), tc.want)) {
				t.Errorf("Plan = %v, want %q", err, tc.want)
			}
		})
	}
}
