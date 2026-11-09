package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

func newSigning(status wsStatus, listed string) (*Signing, *effecttest.Machine) {
	r, m := newRun("lan", status, false, false)
	g := &Gh{M: m, Out: &effecttest.Out{}, LookPath: found}
	m.Results["/bin/gh auth status -h github.com"] = effecttest.Result{}
	m.Results["/bin/gh ssh-key add /l/signing.pub --type signing --title ws1"] = effecttest.Result{}
	m.Results["/bin/gh api --paginate user/ssh_signing_keys --jq .[].key"] =
		effecttest.Result{Out: []byte(listed)}
	return &Signing{Gh: g, Ws: r.Ws, Lock: "/l"}, m
}

func TestSigningPlan(t *testing.T) {
	key := wsStatus{Signing: "ssh-ed25519 AAA", Hostname: "ws1"}
	for name, tc := range map[string]struct {
		status       wsStatus
		listed, want string
	}{
		"no key":  {wsStatus{Signing: "none"}, "", ""},
		"listed":  {key, "ssh-ed25519 AAA\n", ""},
		"missing": {key, "ssh-ed25519 BBB\n", "--title ws1"},
		"no ssh":  {key, "", "the remote script reads nothing"},
	} {
		s, m := newSigning(tc.status, tc.listed)
		if name == "no ssh" {
			delete(m.Results, remoteLine(s.Ws.remote, "status"))
		}
		p, err := s.Plan(context.Background())
		got := errText(err)
		for _, c := range p.Changes {
			got += c.String()
		}
		if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("%s: Plan = %q; want %q", name, got, tc.want)
		}
	}
}

func TestSigningApply(t *testing.T) {
	s, m := newSigning(wsStatus{Signing: "ssh-ed25519 AAA", Hostname: "ws1"}, "")
	if err := s.Apply(context.Background(), step.Plan{}); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/l/signing.pub"]); got != "ssh-ed25519 AAA\n" {
		t.Errorf("signing.pub = %q", got)
	}
	if !strings.Contains(strings.Join(m.Calls, "\n"), "gh ssh-key add /l/signing.pub --type signing --title ws1") {
		t.Errorf("calls = %q", m.Calls)
	}
}

func TestSigningPlanWaitsOnGhUnderCheck(t *testing.T) {
	s, m := newSigning(wsStatus{Signing: "ssh-ed25519 AAA"}, "")
	m.Results["/bin/gh auth status -h github.com"] = effecttest.Result{Err: errors.New("logged out")}
	out := &effecttest.Out{}
	s.Gh.Out = out
	if _, err := s.Plan(context.Background()); err == nil {
		t.Fatal("Plan without --check passed a gh with no login")
	}
	s.Check = true
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || !strings.Contains(strings.Join(out.Lines, "\n"), "after gh on the Mac") {
		t.Errorf("Plan = %v, %v; out %q", p, err, out.Lines)
	}
}
