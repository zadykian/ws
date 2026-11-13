package wizard

import (
	guided "github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/answers"
	"github.com/zadykian/ws/roles/client/macos/stages/gh"
	"github.com/zadykian/ws/roles/client/macos/stages/keys"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"github.com/zadykian/ws/roles/client/macos/stages/tools"
)

// TestEveryStageStatesItsSide fails where a stage is added with no side, as a screen could not
// put it in a list.
func TestEveryStageStatesItsSide(t *testing.T) {
	want := map[string]guided.Side{
		stage.Update: stage.Mac, stage.Tools: stage.Mac, stage.Homebrew: stage.Mac, stage.Answers: stage.Mac, stage.Key: stage.Mac,
		stage.GitHub: stage.Mac, stage.Setup: stage.Mac, stage.Host: stage.Mac, stage.Login: stage.Mac, stage.Sudo: session.Ws,
		stage.Install: session.Ws, stages.RunStage: session.Ws, session.ClaudeStage: session.Ws, stages.SigningStage: stage.Mac, stage.Tunnel: stage.Mac,
		stage.Cable: stage.Mac, (&stages.WsValues{}).Name(): session.Ws,
	}
	list := []step.Step{&tools.MacUpdate{}, &tools.Tools{}, &tools.Homebrew{}, &answers.Answers{}, &keys.MacKey{}, &gh.Gh{}, &tools.Setup{},
		&HostKey{}, &Login{}, &Sudo{}, &tools.WsInstall{}, &stages.WsRun{}, &stages.WsClaude{}, &stages.Signing{GitHub: &gh.MacGitHub{}}, &Tunnel{}, &Cable{},
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
