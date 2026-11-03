package server

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"testing"
	"time"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/step"
)

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
		pl, err := Packages{M: m}.Plan(context.Background())
		got := ""
		if len(pl.Changes) > 0 {
			got = pl.Changes[0].String()
		}
		if (err != nil) != tt.fail || got != tt.want {
			t.Errorf("%s: got %q, %v; want %q, fail %v", tt.name, got, err, tt.want, tt.fail)
		}
	}
}

func TestPackagesApply(t *testing.T) {
	const update = "env DEBIAN_FRONTEND=noninteractive apt-get update"
	const install = "env DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 install -y " +
		"--no-install-recommends python3-apt"
	pl := step.Plan{Changes: []step.Change{installChange{[]string{"python3-apt"}}}}

	t.Run("update works", func(t *testing.T) {
		m := effecttest.NewMachine()
		m.Results[update], m.Results[install] = effecttest.Result{}, effecttest.Result{}
		var waits []time.Duration
		p := Packages{M: m, Out: &effecttest.Out{}, Sleep: func(d time.Duration) { waits = append(waits, d) }}
		if err := p.Apply(context.Background(), pl); err != nil {
			t.Fatal(err)
		}
		if want := []string{"run " + update, "run " + install}; !slices.Equal(m.Calls, want) || len(waits) != 0 {
			t.Errorf("calls %q, waits %v; want %q", m.Calls, waits, want)
		}
	})
	t.Run("update never works", func(t *testing.T) {
		m := effecttest.NewMachine()
		m.Results[update] = effecttest.Result{Err: errors.New("locked")}
		var waits []time.Duration
		p := Packages{M: m, Out: &effecttest.Out{}, Sleep: func(d time.Duration) { waits = append(waits, d) }}
		if err := p.Apply(context.Background(), pl); err == nil {
			t.Fatal("no error")
		}
		if len(m.Calls) != 60 || len(waits) != 59 || waits[0] != 10*time.Second {
			t.Errorf("%d tries, %d waits; want 60 tries, 59 waits of 10s", len(m.Calls), len(waits))
		}
	})
}
