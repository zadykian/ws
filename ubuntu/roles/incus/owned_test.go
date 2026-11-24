package incus

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/ubuntu/values"
)

const ownedList = "nft -j list set bridge ws_sources owned4"

var twoOwners = []values.Guest{
	{Name: "lab", Address: "10.100.0.10"},
	{Name: "build", Address: "10.100.0.11"},
}

func TestAnEmptySetGetsEachGuestsAddressAndPort(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[ownedList] = effecttest.Result{Out: []byte(`{"nftables":[{"metainfo":{}},{"set":{"family":"bridge","name":"owned4","table":"ws_sources","type":["ipv4_addr","ifname"]}}]}`)}
	m.Results["nft -f -"] = effecttest.Result{}
	changes, err := Owned{M: m, Guests: twoOwners}.Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	want := "add element bridge ws_sources owned4 { 10.100.0.10 . \"ws-lab\", 10.100.0.11 . \"ws-build\" }\n"
	if m.Inputs[0] != want {
		t.Errorf("nft got %q, want %q", m.Inputs[0], want)
	}
}

func TestASetThatHoldsTheGuestsIsLeftAlone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[ownedList] = effecttest.Result{Out: []byte(`{"nftables":[{"set":{"name":"owned4","elem":[{"concat":["10.100.0.11","ws-build"]},{"concat":["10.100.0.10","ws-lab"]}]}}]}`)}
	if changes, err := (Owned{M: m, Guests: twoOwners}).Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("changes = %v, err = %v, want none", changes, err)
	}
}

func TestAnElementOfNoGuestIsDeletedAndAMovedAddressIsReplaced(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[ownedList] = effecttest.Result{Out: []byte(`{"nftables":[{"set":{"name":"owned4","elem":[{"concat":["10.100.0.99","ws-lab"]},{"concat":["10.100.0.11","ws-build"]},{"concat":["10.100.0.50","ws-gone"]}]}}]}`)}
	m.Results["nft -f -"] = effecttest.Result{}
	changes, err := Owned{M: m, Guests: twoOwners}.Plan(t.Context())
	if err != nil || len(changes) != 2 {
		t.Fatalf("changes = %v, err = %v, want an add and a delete", changes, err)
	}
	for _, c := range changes {
		if err := c.Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	if !strings.HasPrefix(m.Inputs[0], "add element") || !strings.Contains(m.Inputs[0], `10.100.0.10 . "ws-lab"`) {
		t.Errorf("first script = %q", m.Inputs[0])
	}
	if want := "delete element bridge ws_sources owned4 { 10.100.0.99 . \"ws-lab\", 10.100.0.50 . \"ws-gone\" }\n"; m.Inputs[1] != want {
		t.Errorf("second script = %q, want %q", m.Inputs[1], want)
	}
}

func TestATableThatIsNotLoadedFailsTheRun(t *testing.T) {
	m := effecttest.NewMachine()
	if _, err := (Owned{M: m, Guests: twoOwners}).Plan(t.Context()); err == nil {
		t.Errorf("plan found no error where nft could not list the set")
	}
}
