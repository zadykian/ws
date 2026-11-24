package setup_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/internal/setup"
)

func TestPlanStepsSaysEachChangeAndWritesNothing(t *testing.T) {
	m := effecttest.NewMachine()
	var out effecttest.Out
	steps := []resource.Resource{file.Dir{M: m, Path: "/etc/mise", Mode: 0o755}, file.Dir{M: m, Path: "/opt", Mode: 0o755}}
	m.Dirs["/opt"] = 0o755
	due, err := setup.PlanSteps(t.Context(), &out, steps)
	if err != nil || !due {
		t.Fatalf("due = %v, err = %v; want a change due", due, err)
	}
	if want := []string{"say: would create directory /etc/mise, owner 0:0, mode 755"}; !slices.Equal(out.Lines, want) {
		t.Errorf("said %q, want %q", out.Lines, want)
	}
	if _, ok := m.Dirs["/etc/mise"]; ok {
		t.Error("the plan made the directory")
	}
	out.Lines = nil
	if due, err := setup.PlanSteps(t.Context(), &out, steps[1:]); due || err != nil || len(out.Lines) != 0 {
		t.Errorf("a step with nothing to do: due = %v, err = %v, said %q", due, err, out.Lines)
	}
}
