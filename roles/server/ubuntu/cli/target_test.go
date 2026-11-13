package cli

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
)

func TestRunOfNamesEveryStepOfARunAndLeavesPrepareOutOfTheList(t *testing.T) {
	r := roleOf("a", func(effect.Machine, effect.Out) step.Step { return stepOf{name: "one"} })
	cfg := role.Config{M: effecttest.NewMachine(), Prepare: []step.Step{stepOf{name: "first"}}}
	run := runOf(cfg, []role.Role{r})
	got := run.Groups(false)
	if run.Listed != 1 || len(got) != 2 || got[0].Name != role.Prepare || !slices.Equal(got[0].Steps, []string{"first"}) ||
		got[1].Name != "a" || !slices.Equal(got[1].Steps, []string{"one"}) {
		t.Errorf("listed = %d, groups = %+v", run.Listed, got)
	}
}
