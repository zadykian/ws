package main

import (
	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/step"
)

// wizardQueue returns the stages as the lists show them, each on its machine. ws mac apply's steps
// stand under their stage, and ws's roles too, by name, until ws reports their steps. Cable waits on
// ws mac apply here alone: it applies those steps itself, and a list must hold them on beside it,
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
