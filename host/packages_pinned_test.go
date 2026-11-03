package host

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestPackagesPlanPinnedVersion(t *testing.T) {
	exit1 := exec.Command("sh", "-c", "exit 1").Run()
	tests := []struct {
		name string
		res  effecttest.Result
		want string
		fail bool
	}{
		{"the version matches", effecttest.Result{Out: []byte("24.1.0-1nodesource1 ii\n")}, "", false},
		{"another major version", effecttest.Result{Out: []byte("22.9.0-1nodesource1 ii\n")},
			"install nodejs=24.*", false},
		{"removed with config left", effecttest.Result{Out: []byte("24.1.0 rc\n")}, "install nodejs=24.*", false},
		{"unknown", effecttest.Result{Err: exit1}, "install nodejs=24.*", false},
		{"no command", effecttest.Result{Err: errors.New("not found")}, "", true},
	}
	for _, tt := range tests {
		m := effecttest.NewMachine()
		m.Results["dpkg-query -W -f ${Version} ${db:Status-Abbrev}\n nodejs"] = tt.res
		pl, err := Packages{M: m, Names: []string{"nodejs=24.*"}}.Plan(context.Background())
		got := ""
		if len(pl) > 0 {
			got = pl[0].String()
		}
		if (err != nil) != tt.fail || got != tt.want {
			t.Errorf("%s: got %q, %v; want %q, fail %v", tt.name, got, err, tt.want, tt.fail)
		}
	}
}
