package cli

import (
	"context"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/server/ubuntu"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
)

// localSide names the machine of a stage that names none. The lists draw ws's stages under the
// Mac's name, as the Mac's wizard does.
const localSide wizard.Side = "Mac"

// wizardSide is ws server ubuntu wizard's side. Every stage runs on ws, with no caffeinate.
func wizardSide(p enginecli.Env) engine.Side {
	return engine.Side{
		Stages: func(e engine.WizardEffects, o engine.WizardOptions) []step.Step {
			return serverWizardStages(p, e, o)
		},
		Local: localSide, Nested: NestedRows, WsRun: WsRunSide(p),
		Read: func(string, any) (tui.StateReader, func(context.Context)) {
			return StateRead(serverOpen, p.Version), nil
		},
	}
}

// WsRunSide finds the stage of ws's run in a list, with what the lists ask of it.
func WsRunSide(p enginecli.Env) func([]step.Step) (engine.WsRun, bool) {
	return func(list []step.Step) (engine.WsRun, bool) {
		run := wsRunOf(list)
		if run == nil {
			return engine.WsRun{}, false
		}
		return engine.WsRun{Name: run.Name(), Await: run.WaitsOn(), State: func(ctx context.Context) ([]role.RoleState, error) {
			return run.Ws.State(ctx, p.Version)
		}}, true
	}
}

// NestedRows returns the groups that ws's run stage runs, as the lists show them under it, or nil
// for another stage.
func NestedRows(stage string) []tui.Group {
	var groups []tui.Group
	if stage == (&stages.WsRun{}).Name() {
		for _, name := range ubuntu.Names() {
			groups = append(groups, tui.Group{Name: name})
		}
	}
	return groups
}

// wsRunOf returns the stage of ws's run, or nil where the list holds none.
func wsRunOf(list []step.Step) *stages.WsRun {
	for _, s := range list {
		if run, ok := s.(*stages.WsRun); ok {
			return run
		}
	}
	return nil
}
