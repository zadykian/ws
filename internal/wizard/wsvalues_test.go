package wizard

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// newValues returns the stage over host_vars' text.
func newValues(vars string) *WsValues {
	m := effecttest.NewMachine()
	m.Files["/etc/ws/host_vars.yml"] = []byte(vars)
	return &WsValues{Vars: "/etc/ws/host_vars.yml", M: m}
}

func TestWsValuesPlanNamesWhatHostVarsLacks(t *testing.T) {
	for vars, want := range map[string]string{
		"":                                      "would ask security_outbound and ws_user, and name each in /etc/ws/host_vars.yml",
		"ws_user: a\n":                          "would ask security_outbound,",
		"security_outbound: true\n":             "would ask ws_user,",
		"security_outbound: true\nws_user: a\n": "",
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

const (
	targetQuestion = "ws's target: lan for a machine on the LAN or a VM on the Mac, cloud for a cloud VM (lan, cloud)"
	userQuestion   = "ws_user, the user that ws is set up for"
)

// valuesApply applies the stage's plan over host_vars' text, with a terminal where answers are
// given, and returns the keys it wrote, as KEY=VALUE.
func valuesApply(t *testing.T, vars, user string, answers map[string]string) ([]string, error) {
	v := newValues(vars)
	v.User = user
	var set []string
	v.Tty, v.P = answers != nil, &effecttest.Prompt{T: t, Answers: answers}
	v.Set = func(key, value string) error { set = append(set, key+"="+value); return nil }
	p, err := v.Plan(context.Background())
	if err != nil {
		return nil, err
	}
	return set, v.Apply(context.Background(), p)
}

// Each key is written as answered, the defaults too. The defaults are sudo's user on the LAN, root
// at a cloud VM's root login, and root where host_vars names the table off already.
func TestWsValuesWritesEachAnswer(t *testing.T) {
	for name, tc := range map[string]struct {
		vars, user, target string
		want               []string
	}{
		"lan":        {"", "alice", "", []string{"security_outbound=true", "ws_user=alice"}},
		"cloud root": {"", "", "cloud", []string{"security_outbound=false", "ws_user=root"}},
		"named off":  {"security_outbound: false\n", "root", "", []string{"ws_user=root"}},
		"user named": {"ws_user: bob\n", "alice", "cloud", []string{"security_outbound=false"}},
	} {
		set, err := valuesApply(t, tc.vars, tc.user, map[string]string{targetQuestion: tc.target, userQuestion: ""})
		if err != nil || !slices.Equal(set, tc.want) {
			t.Errorf("%s: wrote %q, %v; want %q", name, set, err, tc.want)
		}
	}
}

func TestWsValuesWithNoTerminalNamesTheCommand(t *testing.T) {
	set, err := valuesApply(t, "ws_user: a\n", "", nil)
	if err == nil || !strings.Contains(err.Error(), "sudo ws server ubuntu values set security_outbound VALUE") || set != nil {
		t.Errorf("Apply = %v; wrote %q", err, set)
	}
}

func TestUserRefusal(t *testing.T) {
	for _, tc := range []struct {
		user      string
		cloud, ok bool
	}{{"alice", false, true}, {"root", true, true}, {"root", false, false}, {"", true, false}, {"x y", true, false}} {
		if err := userRefusal(tc.user, tc.cloud); (err == nil) != tc.ok {
			t.Errorf("userRefusal(%q, %v) = %v", tc.user, tc.cloud, err)
		}
	}
}

// ws_user offers sudo's user, else root on a cloud VM alone.
func TestWsUserOffered(t *testing.T) {
	for _, tc := range []struct {
		user, want string
		cloud      bool
	}{{"alice", "alice", false}, {"alice", "alice", true}, {"", "root", true}, {"root", "", false}, {"", "", false}} {
		if got := (&WsValues{User: tc.user}).offered(tc.cloud); got != tc.want {
			t.Errorf("offered(%q, %v) = %q, want %q", tc.user, tc.cloud, got, tc.want)
		}
	}
}
