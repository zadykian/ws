package git

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/account"
)

func TestAllowedSignersTrustTheKeyOnce(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/home/ws/.config"] = 0o700
	a := account.Account{User: account.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000}
	const key = "/home/ws/.ssh/k"
	rs := AllowedSigners(m, a, key, "/home/ws/.config/git/allowed_signers", "me@x.dev")
	run := func() []string {
		var shown []string
		for _, r := range rs {
			changes, err := r.Plan(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range changes {
				shown = append(shown, c.String())
				if err := c.Apply(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
		}
		return shown
	}
	if got := run(); got != nil {
		t.Errorf("without the key: %q, want nothing", got)
	}
	m.Files[key+".pub"] = []byte("ssh-ed25519 AAAA comment\n")
	if got := run(); len(got) != 1 || !strings.Contains(got[0], "me@x.dev ssh-ed25519 AAAA\n") {
		t.Errorf("with the key: %q", got)
	}
	if got := run(); got != nil || m.Dirs["/home/ws/.config"] != 0o700 {
		t.Errorf("second run: %q, .config mode %o", got, m.Dirs["/home/ws/.config"])
	}
	m.Files[key+".pub"] = []byte("-----BEGIN\nx\n")
	if _, err := rs[len(rs)-1].Plan(context.Background()); err == nil {
		t.Error("a private key is taken for a public one")
	}
}
