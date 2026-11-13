package role

import (
	"context"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

func TestStateReadsNoPrepareWhereAConfigHasNone(t *testing.T) {
	var done bool
	r := Role{Name: "a", Steps: func(effect.Machine, effect.Units, effect.Out) []step.Step {
		return []step.Step{todo{&done}}
	}}
	got, err := State(context.Background(), Config{M: effecttest.NewMachine(), Out: &effecttest.Out{}}, []Role{r})
	if err != nil || len(got) != 1 || got[0].Role != "a" {
		t.Errorf("got %+v, err = %v; want the role a alone", got, err)
	}
}
