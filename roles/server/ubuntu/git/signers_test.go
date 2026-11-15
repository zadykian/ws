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
	a := account.Account{User: account.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000}
	const key = "/home/ws/.ssh/k.pub"
	rs := AllowedSigners(m, a, key)
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
	m.Files[key] = []byte("ssh-ed25519 AAAA comment\n")
	if got := run(); len(got) != 2 || !strings.Contains(got[1], "me@zadykian.dev ssh-ed25519 AAAA\n") {
		t.Errorf("with the key: %q, want /etc/ws and the file", got)
	}
	if got := run(); got != nil {
		t.Errorf("second run: %q", got)
	}
	if a := m.Owners["/etc/ws/allowed_signers"]; a != [2]int{} || m.Dirs["/etc/ws"] != 0o755 {
		t.Errorf("owner %v, /etc/ws mode %o; want root's, 755", a, m.Dirs["/etc/ws"])
	}
	m.Files[key] = []byte("-----BEGIN\nx\n")
	if _, err := rs[len(rs)-1].Plan(context.Background()); err == nil {
		t.Error("a private key is taken for a public one")
	}
}
