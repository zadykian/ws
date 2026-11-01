package host_test

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func TestUserRunsWithItsHome(t *testing.T) {
	cases := []struct {
		user host.User
		want string
	}{
		{host.User{Name: "root", Home: "/root"}, "run env HOME=/root git clone -- a b"},
		{host.User{Name: "ws", Home: "/home/ws"}, "root runuser -u ws -- env HOME=/home/ws git clone -- a b"},
	}
	for _, c := range cases {
		m := effecttest.NewMachine()
		line := c.want[len("root "):]
		if c.user.Name == "root" {
			line = c.want[len("run "):]
		}
		m.Results[line] = effecttest.Result{Out: []byte("ok")}
		if err := c.user.Pass(context.Background(), m, "git", "clone", "--", "a", "b"); err != nil {
			t.Fatal(err)
		}
		out, err := c.user.Output(context.Background(), m, "git", "clone", "--", "a", "b")
		if err != nil || string(out) != "ok" {
			t.Errorf("%s: output %q, %v", c.user.Name, out, err)
		}
		if want := []string{c.want, c.want}; !slices.Equal(m.Calls, want) {
			t.Errorf("%s: calls %q, want %q", c.user.Name, m.Calls, want)
		}
	}
}
