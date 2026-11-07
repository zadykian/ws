package wizard

import (
	"context"
	"slices"
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

const targetQuestion = "ws's target: lan for a machine on the LAN or a VM on the Mac, cloud for a cloud VM (lan, cloud)"

// valuesApply applies the stage's plan over host_vars' text, with a terminal where answers are
// given, and returns the keys it wrote, as KEY=VALUE.
func valuesApply(t *testing.T, vars string, answers map[string]string) ([]string, error) {
	v := newValues(vars)
	var set []string
	v.Tty, v.P = answers != nil, &effecttest.Prompt{T: t, Answers: answers}
	v.Set = func(key, value string) error { set = append(set, key+"="+value); return nil }
	p, err := v.Plan(context.Background())
	if err != nil {
		return nil, err
	}
	return set, v.Apply(context.Background(), p)
}

// The target is written as answered, the default too.
func TestWsValuesWritesTheTarget(t *testing.T) {
	for target, want := range map[string]string{"": "security_outbound=true", "cloud": "security_outbound=false"} {
		set, err := valuesApply(t, "", map[string]string{targetQuestion: target})
		if err != nil || !slices.Equal(set, []string{want}) {
			t.Errorf("%q: wrote %q, %v; want %s", target, set, err, want)
		}
	}
}

func TestWsValuesWithNoTerminalNamesTheCommand(t *testing.T) {
	set, err := valuesApply(t, "", nil)
	if err == nil || !strings.Contains(err.Error(), "sudo ws server values set security_outbound VALUE") || set != nil {
		t.Errorf("Apply = %v; wrote %q", err, set)
	}
}
