package git

import (
	"context"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host/account"
)

func TestSigningKeyIsMadeOnceAsTheUser(t *testing.T) {
	m := effecttest.NewMachine()
	a := account.Account{User: account.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000}
	const key = "/home/ws/.ssh/id_ed25519_signing"
	m.Results["runuser -u ws -- env -C /home/ws HOME=/home/ws ssh-keygen -q -t ed25519 -N  -f "+key] = effecttest.Result{}
	var shown []string
	for range 2 {
		for _, r := range SigningKey(m, a, key) {
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
			if _, ok := r.(keygen); ok && len(changes) == 1 {
				m.Files[key] = []byte("private")
			}
		}
	}
	if len(shown) != 2 || m.Dirs["/home/ws/.ssh"] != 0o700 {
		t.Errorf("shown %q, directory mode %o; want the directory and the key once", shown, m.Dirs["/home/ws/.ssh"])
	}
}

// A key in the home itself needs no directory: a Dir at the home would be no path below it.
func TestSigningKeyInTheHomePlansNoDirectory(t *testing.T) {
	a := account.Account{User: account.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000}
	rs := SigningKey(effecttest.NewMachine(), a, "/home/ws/id_ed25519_signing")
	if _, ok := rs[0].(keygen); len(rs) != 1 || !ok {
		t.Errorf("resources %v; want the key alone", rs)
	}
}
