package main

import (
	"slices"

	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/step"
)

// wizardQueue returns the stages as the lists show them, each on its machine. ws mac apply's steps
// stand under their stage, and ws's roles too, by name, until ws reports their steps. Cable waits on
// ws mac apply here alone. It applies those steps itself, and a list must hold them on beside it,
// while a plain run's Cable reads Homebrew as ever.
func wizardQueue(p process, stages []step.Step) []tui.Stage {
	queue := make([]tui.Stage, len(stages))
	for i, s := range stages {
		queue[i] = tui.Stage{Name: s.Name(), Waits: s.WaitsOn()}
		if m, ok := wizard.MachineOf(s); ok && m == wizard.OnWs {
			queue[i].Side = tui.WsSide
		}
		switch st := s.(type) {
		case *wizard.Cable:
			queue[i].Waits = st.Setup.Name()
		case *wizard.Setup:
			queue[i].Nested = []tui.Group{{Name: s.Name(), Steps: setupStepNames(p)}}
		case *wizard.WsRun:
			for _, r := range roles.List() {
				queue[i].Nested = append(queue[i].Nested, tui.Group{Name: r.Name})
			}
		}
	}
	return queue
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
