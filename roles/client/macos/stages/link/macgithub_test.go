package link

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/gh"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
)

func found(name string) (string, error) { return "/bin/" + name, nil }

func newMacSigning(status session.Status, listed string) (*stages.Signing, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\nWS_USER=ada\n")
	out := &effecttest.Out{}
	ws := macSession(m, remote.SSH{Host: "ws", M: m, Lock: "/l", Ways: host.MacWays{M: m, Conf: "/h/mac.conf"}}, out, status, nil)
	g := gh.CLI{M: m, Out: &effecttest.Out{}, LookPath: found}
	m.Results["/bin/gh auth status -h github.com"] = effecttest.Result{}
	m.Results["/bin/gh ssh-key add /l/signing.pub --type signing --title ws1"] = effecttest.Result{}
	m.Results["/bin/gh api --paginate user/ssh_signing_keys --jq .[].key"] =
		effecttest.Result{Out: []byte(listed)}
	return &stages.Signing{Ws: ws, GitHub: &gh.MacGitHub{Gh: g, Lock: "/l"}}, m
}

func TestMacGitHubAddsTheSigningKey(t *testing.T) {
	s, m := newMacSigning(session.Status{Signing: "ssh-ed25519 AAA", Hostname: "ws1"}, "")
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

func TestMacGitHubWaitsOnGhUnderCheck(t *testing.T) {
	s, m := newMacSigning(session.Status{Signing: "ssh-ed25519 AAA"}, "")
	m.Results["/bin/gh auth status -h github.com"] = effecttest.Result{Err: errors.New("logged out")}
	out, g := &effecttest.Out{}, s.GitHub.(*gh.MacGitHub)
	g.Gh.Out = out
	if _, err := s.Plan(context.Background()); err == nil {
		t.Fatal("Plan without --check passed a gh with no login")
	}
	g.Check = true
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || !strings.Contains(strings.Join(out.Lines, "\n"), "after gh on the Mac") {
		t.Errorf("Plan = %v, %v; out %q", p, err, out.Lines)
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
