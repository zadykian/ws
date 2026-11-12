package apply

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	roles "github.com/zadykian/ws/roles/server/ubuntu"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func TestRunAndStateHandEachRoleTheUnits(t *testing.T) {
	u := &effecttest.Units{}
	var got []effect.Units
	r := roles.Role{Name: "a", Steps: func(_ values.Values, _ secrets.Secrets, _ effect.Machine, u effect.Units, _ effect.Out) []step.Step {
		got = append(got, u)
		return nil
	}}
	c := Config{M: effecttest.NewMachine(), U: u, Out: &effecttest.Out{}}
	if _, err := Run(t.Context(), c, []roles.Role{r}); err != nil {
		t.Fatal(err)
	}
	if _, err := State(t.Context(), c, []roles.Role{r}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != effect.Units(u) || got[1] != effect.Units(u) {
		t.Errorf("units handed = %v, want %v twice", got, u)
	}
}
