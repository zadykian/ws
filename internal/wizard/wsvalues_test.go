package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// newValues returns the stage over host_vars' text.
func newValues(vars string) *WsValues {
	m := effecttest.NewMachine()
	m.Files["/etc/ws/host_vars.yml"] = []byte(vars)
	return &WsValues{Vars: "/etc/ws/host_vars.yml", M: m}
}

func TestWsValuesPlanNamesWhatHostVarsLacks(t *testing.T) {
	for vars, want := range map[string]string{
		"ws_user: a\n":              "would ask security_outbound, and name each in /etc/ws/host_vars.yml",
		"":                          "would ask security_outbound,",
		"security_outbound: true\n": "",
	} {
		p, err := newValues(vars).Plan(context.Background())
		got := ""
		if len(p.Changes) > 0 {
			got = p.Changes[0].String()
		}
		if err != nil || (want == "") != (got == "") || !strings.Contains(got, want) {
			t.Errorf("%q: Plan = %q, %v; want %q", vars, got, err, want)
		}
	}
}
