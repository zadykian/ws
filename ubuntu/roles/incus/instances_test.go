package incus

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/ubuntu/values"
)

func TestAStoppedGuestIsStartedAndARunningOneIsLeftAlone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(`[{"name":"a","status":"Stopped"},{"name":"lab","status":"Stopped"},{"name":"b","status":"Running"}]`)}
	m.Results["incus start lab"] = effecttest.Result{}
	changes, err := Running{M: m, Name: "lab"}.Plan(t.Context())
	if err != nil || len(changes) != 1 || changes[0].String() != "start the guest lab" {
		t.Fatalf("changes = %v, err = %v, want the start of lab alone", changes, err)
	}
	if err := changes[0].Apply(t.Context()); err != nil || !slices.Contains(m.Calls, "run incus start lab") {
		t.Errorf("apply = %v, calls %q", err, m.Calls)
	}
	for _, name := range []string{"b", "missing"} {
		if changes, err := (Running{M: m, Name: name}).Plan(t.Context()); err != nil || len(changes) != 0 {
			t.Errorf("%s: changes = %v, err = %v, want none", name, changes, err)
		}
	}
}

func TestStraysAreTheInstancesOfWsThatTheListLacks(t *testing.T) {
	all := []Instance{
		{Name: "lab", Profiles: []string{"ws"}}, {Name: "old", Profiles: []string{"ws"}},
		{Name: "docker-box", Profiles: []string{"default"}},
	}
	got := Strays(all, []values.Guest{{Name: "lab"}})
	if !slices.Equal(got, []string{"old"}) {
		t.Errorf("strays = %q, want old alone: another profile's instance is no guest of ws's", got)
	}
}
