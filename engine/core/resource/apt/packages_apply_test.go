package apt

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestPackagesApply(t *testing.T) {
	const update = "env DEBIAN_FRONTEND=noninteractive apt-get update"
	const install = "env DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold install -y " +
		"--no-install-recommends python3-apt"

	t.Run("update works", func(t *testing.T) {
		m := effecttest.NewMachine()
		m.Results[update], m.Results[install] = effecttest.Result{}, effecttest.Result{}
		var waits []time.Duration
		p := Packages{M: m, Out: &effecttest.Out{}, Sleep: func(d time.Duration) { waits = append(waits, d) }}
		if err := (installChange{p, []string{"python3-apt"}}).Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
		if want := []string{"run " + update, "run " + install}; !slices.Equal(m.Calls, want) || len(waits) != 0 {
			t.Errorf("calls %q, waits %v; want %q", m.Calls, waits, want)
		}
	})
	t.Run("recommends", func(t *testing.T) {
		const withRecommends = "env DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold install -y python3-apt"
		m := effecttest.NewMachine()
		m.Results[update], m.Results[withRecommends] = effecttest.Result{}, effecttest.Result{}
		p := Packages{M: m, Out: &effecttest.Out{}, Recommends: true}
		if err := (installChange{p, []string{"python3-apt"}}).Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
		if want := []string{"run " + update, "run " + withRecommends}; !slices.Equal(m.Calls, want) {
			t.Errorf("calls %q; want %q", m.Calls, want)
		}
	})
	t.Run("update never works", func(t *testing.T) {
		m := effecttest.NewMachine()
		m.Results[update] = effecttest.Result{Err: errors.New("locked")}
		var waits []time.Duration
		p := Packages{M: m, Out: &effecttest.Out{}, Sleep: func(d time.Duration) { waits = append(waits, d) }}
		if err := (installChange{p, []string{"python3-apt"}}).Apply(context.Background()); err == nil {
			t.Fatal("no error")
		}
		if len(m.Calls) != 60 || len(waits) != 59 || waits[0] != 10*time.Second {
			t.Errorf("%d tries, %d waits; want 60 tries, 59 waits of 10s", len(m.Calls), len(waits))
		}
	})
}
