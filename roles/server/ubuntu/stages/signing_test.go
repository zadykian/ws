package stages

import (
	"context"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	guided "github.com/zadykian/ws/engine/core/wizard"
)

// listedGitHub is a GitHub that lists the keys of its field.
type listedGitHub struct {
	listed string
	added  []string
}

func (*listedGitHub) Side() guided.Side { return guided.Side("test") }

func (g *listedGitHub) Lacks(_ context.Context, ws *session.Session, line string) (step.Change, error) {
	if strings.Contains(g.listed, line) {
		return nil, nil
	}
	return signingAdd{ws.Known().Hostname}, nil
}

func (g *listedGitHub) Add(_ context.Context, _ *session.Session, line string) error {
	g.added = append(g.added, line)
	return nil
}

type signingAdd struct{ title string }

func (c signingAdd) String() string { return SigningStage + ": would add --title " + c.title }

func newSigning(status session.Status, listed string) (*Signing, *effecttest.Machine) {
	r, m := newRun("lan", status, false, false)
	return &Signing{Ws: r.Ws, GitHub: &listedGitHub{listed: listed}}, m
}

func TestSigningPlan(t *testing.T) {
	key := session.Status{Signing: "ssh-ed25519 AAA", Hostname: "ws1"}
	for name, tc := range map[string]struct {
		status       session.Status
		listed, want string
	}{
		"no key":  {session.Status{Signing: "none"}, "", ""},
		"listed":  {key, "ssh-ed25519 AAA\n", ""},
		"missing": {key, "ssh-ed25519 BBB\n", "--title ws1"},
		"no ssh":  {key, "", "the remote script reads nothing"},
	} {
		s, m := newSigning(tc.status, tc.listed)
		if name == "no ssh" {
			delete(m.Results, sessiontest.Op("status"))
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
	s, _ := newSigning(session.Status{Signing: "ssh-ed25519 AAA", Hostname: "ws1"}, "")
	if err := s.Apply(context.Background(), step.Plan{}); err != nil {
		t.Fatal(err)
	}
	if got := s.GitHub.(*listedGitHub).added; len(got) != 1 || got[0] != "ssh-ed25519 AAA" {
		t.Errorf("added = %q", got)
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
