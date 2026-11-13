package ufw_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/internal/ufw"
)

const comment = "DNS"

func dns(iface, to string) ufw.Rule {
	return ufw.Rule{Action: "allow", Proto: "any", ToPort: "53", To: to, FromPort: "any", From: "any", Dir: "in", Interface: iface}
}

func userRule(iface, to, from, extra string) string {
	return "### tuple ### allow " + extra + " 53 " + to + " any " + from + " in_" + iface +
		" comment=444e53\n"
}

func plan(t *testing.T, r ufw.Rules) []string {
	changes, err := r.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range changes {
		got = append(got, c.String())
	}
	return got
}

func TestRulesAddTheMissingAndDeleteTheStale(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[ufw.UserRules] = []byte(userRule("docker0", "172.17.0.1", ufw.Any, "any") +
		userRule("docker0", "172.17.0.1", "10.0.0.0/8", "any"))
	r := ufw.Rules{M: m, Comment: comment,
		Want: []ufw.Rule{dns("docker0", "172.17.0.1"), dns("br-+", "172.17.0.1")}}
	want := []string{
		"ufw allow in on br-+ from any to 172.17.0.1 port 53",
		"ufw delete allow in on docker0 from 10.0.0.0/8 to 172.17.0.1 port 53",
	}
	if got := plan(t, r); !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q", got, want)
	}
}

func TestRulesLeaveTheRulesOwnsRefuses(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[ufw.UserRules] = []byte(userRule("docker0", "172.17.0.1", "10.0.0.0/8", "any"))
	r := ufw.Rules{M: m, Comment: comment, Owns: func(r ufw.Rule) bool { return r.From == ufw.Any }}
	if got := plan(t, r); got != nil {
		t.Errorf("plan = %q, want none: the rule is not the role's", got)
	}
}

func TestRulesRefuseARuleWithoutDirection(t *testing.T) {
	bad := dns("docker0", "1.1.1.1")
	bad.Dir = ""
	r := ufw.Rules{M: effecttest.NewMachine(), Comment: comment, Want: []ufw.Rule{bad}}
	if _, err := r.Plan(t.Context()); err == nil {
		t.Error("a rule with no direction passed")
	}
}

func TestRulesSkipWithoutSystemd(t *testing.T) {
	out := &effecttest.Out{}
	r := ufw.Rules{M: effecttest.NewMachine(), Out: out, Comment: comment, NeedsSystemd: true,
		Want: []ufw.Rule{dns("docker0", "1.1.1.1")}}
	if got := plan(t, r); got != nil || len(out.Lines) != 1 {
		t.Errorf("plan = %q, said %q; want a skip said once", got, out.Lines)
	}
}
