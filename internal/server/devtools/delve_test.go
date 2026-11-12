package devtools

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/host"
)

const delveRead = goBin + " version -m /root/go/bin/dlv"

func newDelve(m *effecttest.Machine) delveTool {
	return delveTool{M: m, User: host.User{Name: "root", Home: "/root"}, Path: "/root/go/bin/dlv",
		Version: dlvVersion, GoVersion: "1.27.3"}
}

func TestDelvePlan(t *testing.T) {
	current := "/root/go/bin/dlv: go1.27.3\n\tmod\tgithub.com/go-delve/delve\tv" + dlvVersion + "\th1:abc=\n"
	cases := map[string]struct {
		result effecttest.Result
		want   int
	}{
		"current":  {effecttest.Result{Out: []byte(current)}, 0},
		"older Go": {effecttest.Result{Out: []byte("/root/go/bin/dlv: go1.26.0\n")}, 1},
		"missing":  {effecttest.Result{Err: context.DeadlineExceeded}, 1},
	}
	for name, c := range cases {
		m := effecttest.NewMachine()
		m.Results[delveRead] = c.result
		got, err := newDelve(m).Plan(context.Background())
		if err != nil || len(got) != c.want {
			t.Errorf("%s: %d changes, %v; want %d", name, len(got), err, c.want)
		}
	}
}

func TestDelveBuildsThroughGoInstall(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[delveRead] = effecttest.Result{Err: context.Canceled}
	build := "env HOME=/root env GOBIN=/root/go/bin GOTOOLCHAIN=local GOENV=off GOSUMDB=sum.golang.org " +
		goBin + " install github.com/go-delve/delve/cmd/dlv@v" + dlvVersion
	m.Results[build] = effecttest.Result{}
	m.Files[goBin] = nil
	changes, err := newDelve(m).Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("%v, %v", changes, err)
	}
	if err := changes[0].Apply(context.Background()); err != nil || !slices.Contains(m.Calls, "run "+build) {
		t.Errorf("%v, calls %q", err, m.Calls)
	}
}
