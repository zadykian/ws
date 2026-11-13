package wizard

import (
	guided "github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
)

// TestEveryStageStatesItsSide fails where a stage is added with no side, as a screen could not
// put it in a list.
func TestEveryStageStatesItsSide(t *testing.T) {
	want := map[string]guided.Side{
		macUpdateStage: OnMac, toolsStage: OnMac, homebrewStage: OnMac, answers: OnMac, macKey: OnMac,
		ghStage: OnMac, setupStage: OnMac, hostStage: OnMac, loginStage: OnMac, sudoStage: session.Ws,
		installStage: session.Ws, stages.RunStage: session.Ws, session.ClaudeStage: session.Ws, stages.SigningStage: OnMac, tunnelStage: OnMac,
		cableStage: OnMac, (&stages.WsValues{}).Name(): session.Ws,
	}
	list := []step.Step{&MacUpdate{}, &Tools{}, &Homebrew{}, &Answers{}, &MacKey{}, &Gh{}, &Setup{},
		&HostKey{}, &Login{}, &Sudo{}, &WsInstall{}, &stages.WsRun{}, &stages.WsClaude{}, &stages.Signing{GitHub: &MacGitHub{}}, &Tunnel{}, &Cable{},
		&stages.WsValues{}}
	if len(list) != len(want) {
		t.Fatalf("%d stages, %d wanted", len(list), len(want))
	}
	for _, s := range list {
		got, ok := guided.SideOf(s)
		if !ok {
			t.Errorf("%s states no side", s.Name())
			continue
		}
		if got != want[s.Name()] {
			t.Errorf("%s is on %s, want %s", s.Name(), got, want[s.Name()])
		}
	}
}
