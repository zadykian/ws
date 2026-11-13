package cmd

import (
	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/core/wizard"
	"slices"

	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
)

// wizardQueue returns the stages as the lists show them, each on its machine, with the stage that
// picking it locks. ws client macos apply's steps stand under their stage, and ws's roles too, by
// name, until ws reports their steps.
func wizardQueue(side engine.Side, stages []step.Step) []tui.Stage {
	queue := make([]tui.Stage, len(stages))
	for i, s := range stages {
		queue[i] = tui.Stage{Name: s.Name(), Side: string(side.Local), Waits: wizard.Locks(s), Nested: side.Nested(s.Name())}
		if m, ok := wizard.SideOf(s); ok {
			queue[i].Side = string(m)
		}
	}
	return queue
}

// wizardLeft is the run's LeftOut for choice. It leaves out a stage that choice doesn't name with
// its steps, and a step of ws client macos apply off in a stage it runs in part. ws client macos
// apply's run asks it of the steps with a row alone.
func wizardLeft(choice tui.QueueChoice) func(parent, name string) bool {
	return func(parent, name string) bool {
		if parent == "" {
			return !slices.Contains(choice.Only, name)
		}
		on, part := choice.Partial[parent]
		if !part {
			return !slices.Contains(choice.Only, parent)
		}
		return !slices.Contains(on, parent+"/"+name)
	}
}

// reportRoles gives the lists what ws says of its roles, or why it says nothing. Prepare's steps
// have no row, as in ws server ubuntu apply's own list. They run whatever the roles do, and a box
// would choose nothing, where --only refuses their name.
func reportRoles(stage string, states []role.RoleState, err error, report func(string, []tui.StateSection, string)) {
	if err != nil {
		report(stage, nil, err.Error())
		return
	}
	report(stage, slices.DeleteFunc(engine.RoleSections(states), func(s tui.StateSection) bool {
		return s.Name == role.Prepare
	}), "")
}
