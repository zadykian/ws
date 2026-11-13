package cmd

import (
	guided "github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
	"slices"

	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/roles/client/macos/setup"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"github.com/zadykian/ws/roles/server/ubuntu"
)

// wizardQueue returns the stages as the lists show them, each on its machine, with the stage that
// picking it locks. ws client macos apply's steps stand under their stage, and ws's roles too, by
// name, until ws reports their steps.
func wizardQueue(stages []step.Step) []tui.Stage {
	queue := make([]tui.Stage, len(stages))
	for i, s := range stages {
		queue[i] = tui.Stage{Name: s.Name(), Side: string(stage.Mac), Waits: guided.Locks(s), Nested: nestedRows(s.Name())}
		if m, ok := guided.SideOf(s); ok {
			queue[i].Side = string(m)
		}
	}
	return queue
}

// nestedRows returns the groups that the stage named runs, as the lists show them under it.
func nestedRows(stage string) []tui.Group {
	var groups []tui.Group
	switch stage {
	case (&wizard.Setup{}).Name():
		groups = []tui.Group{{Name: stage, Steps: setup.Names()}}
	case (&stages.WsRun{}).Name():
		for _, name := range ubuntu.Names() {
			groups = append(groups, tui.Group{Name: name})
		}
	}
	return groups
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
	report(stage, slices.DeleteFunc(roleSections(states), func(s tui.StateSection) bool {
		return s.Name == role.Prepare
	}), "")
}
