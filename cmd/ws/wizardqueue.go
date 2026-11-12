package main

import (
	"slices"

	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/step"
)

// wizardQueue returns the stages as the lists show them, each on its machine, with the stage that
// picking it locks. ws mac apply's steps stand under their stage, and ws's roles too, by name, until
// ws reports their steps.
func wizardQueue(stages []step.Step) []tui.Stage {
	queue := make([]tui.Stage, len(stages))
	for i, s := range stages {
		queue[i] = tui.Stage{Name: s.Name(), Waits: wizard.Locks(s), Nested: nestedRows(s.Name())}
		if m, ok := wizard.MachineOf(s); ok && m == wizard.OnWs {
			queue[i].Side = tui.WsSide
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
	case (&wizard.WsRun{}).Name():
		for _, r := range roles.List() {
			groups = append(groups, tui.Group{Name: r.Name})
		}
	}
	return groups
}

// wizardLeft is the run's LeftOut for choice. It leaves out a stage that choice doesn't name with
// its steps, and a step of ws mac apply off in a stage it runs in part. setup names the steps with
// a row. ssh again has none, as a check has no second plan, so it follows ssh by its wait, as in
// ws mac apply's own list.
func wizardLeft(choice tui.Choice, setup []string) func(parent, name string) bool {
	return func(parent, name string) bool {
		if parent == "" {
			return !slices.Contains(choice.Only, name)
		}
		on, part := choice.Partial[parent]
		if !part {
			return !slices.Contains(choice.Only, parent)
		}
		return slices.Contains(setup, name) && !slices.Contains(on, parent+"/"+name)
	}
}

// reportRoles gives the lists what ws says of its roles, or why it says nothing. Prepare's steps
// have no row, as in ws server apply's own list. They run whatever the roles do, and a box would
// choose nothing, where --only refuses their name.
func reportRoles(stage string, states []apply.RoleState, err error, report func(string, []tui.StateSection, string)) {
	if err != nil {
		report(stage, nil, err.Error())
		return
	}
	report(stage, slices.DeleteFunc(roleSections(states), func(s tui.StateSection) bool {
		return s.Name == apply.Prepare
	}), "")
}
