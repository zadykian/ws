package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/step"
)

const keysCmd = "curl -fsSL --retry 3 --proto =https https://api.github.com/users/ada-1/ssh_signing_keys?per_page=100"

func newWsSigning(status wsStatus, login, listed string) *WsSigning {
	r, m := newRun("lan", status, false, false)
	r.Want = func() (conf.Conf, error) { return conf.Conf{User: "ada"}, nil }
	m.Results[loginCmd] = effecttest.Result{Out: []byte(login)}
	m.Results[keysCmd] = effecttest.Result{Out: []byte(listed)}
	return &WsSigning{Run: r, M: m, Hand: &Hand{}}
}

func TestWsSigningPlan(t *testing.T) {
	key := wsStatus{Signing: "ssh-ed25519 AAA"}
	for name, tc := range map[string]struct {
		status              wsStatus
		login, listed, want string
	}{
		"no key":   {wsStatus{Signing: "none"}, "", "", ""},
		"listed":   {key, "motd\nada-1\n", `[{"id":1,"key":"ssh-ed25519 AAA"}]`, ""},
		"missing":  {key, "ada-1\n", `[{"key":"ssh-ed25519 BBB"}]`, "lists it for no login"},
		"no login": {key, "", "", "names a GitHub login"},
		"no list":  {key, "ada-1\n", "<html>", "curl read no list"},
		"no ssh":   {key, "", "", "the remote script reads nothing"},
	} {
		s := newWsSigning(tc.status, tc.login, tc.listed)
		if name == "no ssh" {
			delete(s.M.(*effecttest.Machine).Results, remoteLine(s.Run.Remote, "status"))
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

// The key goes to what stays by hand, to add on GitHub, and the run plans nothing more of it.
func TestWsSigningApply(t *testing.T) {
	s := newWsSigning(wsStatus{Signing: "ssh-ed25519 AAA"}, "", "")
	if err := s.Apply(context.Background(), step.Plan{}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(s.Hand.lines, "\n"); !strings.HasSuffix(got, "(docs/server-wizard.md): ssh-ed25519 AAA") {
		t.Errorf("hand = %q", got)
	}
	if p, err := s.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("Plan after Apply = %v, %v", p, err)
	}
}
