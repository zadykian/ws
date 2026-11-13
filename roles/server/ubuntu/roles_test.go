package ubuntu

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func TestListHasEachRoleOnce(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range List() {
		if r.Name == "" || seen[r.Name] {
			t.Errorf("role %q is empty or twice in the list", r.Name)
		}
		seen[r.Name] = true
		if r.Steps == nil {
			t.Errorf("role %q has no steps", r.Name)
		}
	}
}

func TestListIsACopy(t *testing.T) {
	l := List()
	l[0].Name = "changed"
	if List()[0].Name == "changed" {
		t.Error("List returns the list itself")
	}
}

func TestZedRunsAfterFresh(t *testing.T) {
	var names []string
	for _, r := range List() {
		names = append(names, r.Name)
	}
	i := slices.Index(names, "zed")
	if i < 1 || names[i-1] != "fresh" {
		t.Errorf("zed is misplaced in %v", names)
	}
}

func TestFreshRunsJustBeforeZed(t *testing.T) {
	var names []string
	for _, r := range List() {
		names = append(names, r.Name)
	}
	i := slices.Index(names, "fresh")
	if i < 1 || names[i-1] != "otelcol" || names[i+1] != "zed" {
		t.Errorf("fresh is misplaced in %v", names)
	}
}

func TestRolesCloseOverTheValuesAndSecrets(t *testing.T) {
	var gotV []values.Values
	var gotS []secrets.Secrets
	record := func(v values.Values, s secrets.Secrets, _ effect.Machine, _ effect.Units, _ effect.Out) []step.Step {
		gotV, gotS = append(gotV, v), append(gotS, s)
		return nil
	}
	list := []Role{{Name: "a", Steps: record}, {Name: "b", Steps: record}}
	v := values.Values{Home: "/home/ws"}
	s := secrets.Secrets{Whole: true, Env: map[string]string{"A": "b"}}
	var names []string
	for _, r := range closeOver(list, v, s) {
		names = append(names, r.Name)
		r.Steps(effecttest.NewMachine(), &effecttest.Units{}, &effecttest.Out{})
	}
	if !slices.Equal(names, []string{"a", "b"}) {
		t.Errorf("names = %q, want a, b in order", names)
	}
	if len(gotV) != 2 || gotV[0].Home != "/home/ws" || gotV[1].Home != "/home/ws" || !gotS[0].Whole || gotS[1].Env["A"] != "b" {
		t.Errorf("roles got values %+v and secrets %+v, want the closed-over ones", gotV, gotS)
	}
}

func TestRolesFollowNames(t *testing.T) {
	var names []string
	for _, r := range Roles(values.Values{}, secrets.Secrets{}) {
		names = append(names, r.Name)
	}
	if !slices.Equal(names, Names()) {
		t.Errorf("Roles are %q, want Names %q", names, Names())
	}
}

func TestNamesAreACopy(t *testing.T) {
	n := Names()
	n[0] = "changed"
	if Names()[0] == "changed" {
		t.Error("Names returns the list itself")
	}
}
