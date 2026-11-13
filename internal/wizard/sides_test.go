package wizard

import (
	guided "github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
)

// TestEveryStageStatesItsSide fails where a stage is added with no side, as a screen could not
// put it in a list.
func TestEveryStageStatesItsSide(t *testing.T) {
	want := map[string]guided.Side{
		macUpdateStage: OnMac, toolsStage: OnMac, homebrewStage: OnMac, answers: OnMac, macKey: OnMac,
		ghStage: OnMac, setupStage: OnMac, hostStage: OnMac, loginStage: OnMac, sudoStage: session.Ws,
		installStage: session.Ws, RunStage: session.Ws, session.ClaudeStage: session.Ws, SigningStage: OnMac, tunnelStage: OnMac,
		cableStage: OnMac, valuesStage: session.Ws,
	}
	stages := []step.Step{&MacUpdate{}, &Tools{}, &Homebrew{}, &Answers{}, &MacKey{}, &Gh{}, &Setup{},
		&HostKey{}, &Login{}, &Sudo{}, &WsInstall{}, &WsRun{}, &WsClaude{}, &Signing{GitHub: &MacGitHub{}}, &Tunnel{}, &Cable{},
		&WsValues{}}
	if len(stages) != len(want) {
		t.Fatalf("%d stages, %d wanted", len(stages), len(want))
	}
	for _, s := range stages {
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
