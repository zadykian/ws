package cmd

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func TestServerRolesCloseOverTheValuesAndSecrets(t *testing.T) {
	var gotV []values.Values
	var gotS []secrets.Secrets
	record := func(v values.Values, s secrets.Secrets, _ effect.Machine, _ effect.Units, _ effect.Out) []step.Step {
		gotV, gotS = append(gotV, v), append(gotS, s)
		return nil
	}
	list := []ubuntu.Role{{Name: "a", Steps: record}, {Name: "b", Steps: record}}
	v := values.Values{Home: "/home/ws"}
	s := secrets.Secrets{Whole: true, Env: map[string]string{"A": "b"}}
	roles := serverRoles(list, v, s)
	var names []string
	for _, r := range roles {
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
