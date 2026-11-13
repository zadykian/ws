package shell

import (
	"context"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host/account"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func TestCheckClones(t *testing.T) {
	for _, c := range []struct {
		repo, dest string
		bad        bool
	}{
		{"https://github.com/a/b", "b", false},
		{"git@github.com:a/b", "b", false},
		{"https://host/a@b", "b", false},
		{"https://user:token@host/a", "b", true},
		{"http://token@host/a", "b", true},
		{"", "b", true},
		{"https://h/a", "", true},
		{"https://h/a", "a*", true},
		{"https://h/a", "a?", true},
		{"https://h/a", "a[0]", true},
		{"https://h/a\nb", "b", true},
	} {
		err := checkClones([]values.GitClone{{Repo: c.repo, Dest: c.dest}})
		if (err != nil) != c.bad {
			t.Errorf("%q into %q: %v", c.repo, c.dest, err)
		}
	}
}

func TestClonesRunOnceAsTheUser(t *testing.T) {
	m := effecttest.NewMachine()
	a := account.Account{User: account.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000}
	m.Results["runuser -u ws -- env -C /home/ws HOME=/home/ws git -C /home/ws clone -- https://h/a work/a"] = effecttest.Result{}
	cl := clones{M: m, A: a, Clones: []values.GitClone{
		{Repo: "https://h/a", Dest: "work/a"}, {Repo: "https://h/b", Dest: "/srv/b"},
	}}
	m.Dirs["/srv/b"] = 0o755
	changes, err := cl.Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("got %v, %v", changes, err)
	}
	if err := changes[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	m.Dirs["/home/ws/work/a"] = 0o755
	if changes, err := cl.Plan(context.Background()); err != nil || len(changes) != 0 {
		t.Errorf("second plan = %v, %v", changes, err)
	}
}
