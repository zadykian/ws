package devtools

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

const goplsRead = "env HOME=/root /root/go/bin/gopls version -json"

func newGopls(m *effecttest.Machine) goplsTool {
	return goplsTool{M: m, User: host.User{Name: "root", Home: "/root"}, Path: "/root/go/bin/gopls",
		Version: "0.23.0", GoVersion: "1.27.3"}
}

func TestGoplsPlan(t *testing.T) {
	cases := []struct {
		name   string
		result effecttest.Result
		want   int
	}{
		{"current", effecttest.Result{Out: []byte(`{"GoVersion":"go1.27.3","Version":"v0.23.0"}`)}, 0},
		{"older gopls", effecttest.Result{Out: []byte(`{"GoVersion":"go1.27.3","Version":"v0.22.0"}`)}, 1},
		{"older Go", effecttest.Result{Out: []byte(`{"GoVersion":"go1.26.0","Version":"v0.23.0"}`)}, 1},
		{"missing", effecttest.Result{Err: context.DeadlineExceeded}, 1},
	}
	for _, c := range cases {
		m := effecttest.NewMachine()
		m.Results[goplsRead] = c.result
		got, err := newGopls(m).Plan(context.Background())
		if err != nil || len(got) != c.want {
			t.Errorf("%s: %d changes, %v; want %d", c.name, len(got), err, c.want)
		}
	}
}

func TestGoplsRefusesAnUnreadableVersion(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[goplsRead] = effecttest.Result{Out: []byte("not json")}
	if _, err := newGopls(m).Plan(context.Background()); err == nil {
		t.Error("an unreadable version is planned")
	}
}

func TestGoplsBuildsWithTheLocalGo(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[goplsRead] = effecttest.Result{Err: context.Canceled}
	build := "env HOME=/root env GOBIN=/root/go/bin GOTOOLCHAIN=local " + goBin + " install golang.org/x/tools/gopls@v0.23.0"
	m.Results[build] = effecttest.Result{}
	g := newGopls(m)
	changes, err := g.Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("%v, %v", changes, err)
	}
	if err := changes[0].Apply(context.Background()); err == nil {
		t.Fatal("gopls is built with no Go")
	}
	m.Files[goBin] = nil
	if err := changes[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Calls, "run "+build) {
		t.Errorf("calls %q", m.Calls)
	}
}
