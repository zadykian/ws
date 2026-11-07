package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/conf"
)

const keysCmd = "curl -fsSL --retry 3 --proto =https https://api.github.com/users/ada-1/ssh_signing_keys?per_page=100"

func newWsSigning(status, login, listed string) *WsSigning {
	r, m := newRun("lan", status, false, false)
	r.Want = func() (conf.Conf, error) { return conf.Conf{User: "ada"}, nil }
	m.Results[loginCmd] = effecttest.Result{Out: []byte(login)}
	m.Results[keysCmd] = effecttest.Result{Out: []byte(listed)}
	return &WsSigning{Run: r, M: m, Hand: &Hand{}}
}

func TestWsSigningPlan(t *testing.T) {
	key := "signing ssh-ed25519 AAA\n"
	for name, tc := range map[string]struct{ status, login, listed, want string }{
		"no key":   {"signing none\n", "", "", ""},
		"listed":   {key, "motd\nada-1\n", `[{"id":1,"key":"ssh-ed25519 AAA"}]`, ""},
		"missing":  {key, "ada-1\n", `[{"key":"ssh-ed25519 BBB"}]`, "lists it for no login"},
		"no login": {key, "", "", "names a GitHub login"},
		"no list":  {key, "ada-1\n", "<html>", "curl read no list"},
		"no ssh":   {"", "", "", "the remote script reads nothing"},
	} {
		s := newWsSigning(tc.status, tc.login, tc.listed)
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
