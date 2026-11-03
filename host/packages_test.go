package host

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

var packages = []string{"ansible-core", "ca-certificates", "python3-apt"}

const queryLine = "dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n ansible-core ca-certificates python3-apt"

func TestMissingPackages(t *testing.T) {
	tests := []struct {
		name, out string
		want      []string
	}{
		{"all there", "ansible-core ii\nca-certificates ii\npython3-apt ii\n", nil},
		{"none listed", "", packages},
		{"removed with config left", "ansible-core ii\nca-certificates rc\npython3-apt ii\n", []string{"ca-certificates"}},
		{"one unknown", "ansible-core ii\npython3-apt ii\n", []string{"ca-certificates"}},
		{"half installed", "ansible-core ii\nca-certificates ii\npython3-apt iU\n", []string{"python3-apt"}},
	}
	for _, tt := range tests {
		if got := missingPackages(tt.out, packages); !slices.Equal(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestPackagesPlan(t *testing.T) {
	exit1 := exec.Command("sh", "-c", "exit 1").Run()
	exit2 := exec.Command("sh", "-c", "exit 2").Run()
	tests := []struct {
		name string
		res  effecttest.Result
		want string
		fail bool
	}{
		{"all", effecttest.Result{Out: []byte("ansible-core ii\nca-certificates ii\npython3-apt ii\n")}, "", false},
		{"unknown package", effecttest.Result{Out: []byte("ansible-core ii\n"), Err: exit1},
			"install ca-certificates python3-apt", false},
		{"a failure", effecttest.Result{Err: exit2}, "", true},
		{"no command", effecttest.Result{Err: errors.New("not found")}, "", true},
	}
	for _, tt := range tests {
		m := effecttest.NewMachine()
		m.Results[queryLine] = tt.res
		pl, err := Packages{M: m, Names: packages}.Plan(context.Background())
		got := ""
		if len(pl) > 0 {
			got = pl[0].String()
		}
		if (err != nil) != tt.fail || got != tt.want {
			t.Errorf("%s: got %q, %v; want %q, fail %v", tt.name, got, err, tt.want, tt.fail)
		}
	}
}
