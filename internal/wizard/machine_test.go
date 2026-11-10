package wizard

import (
	"testing"

	"github.com/zadykian/ws/step"
)

// TestEveryStageNamesItsMachine fails where a stage is added with no machine, as a screen could not
// put it in a list.
func TestEveryStageNamesItsMachine(t *testing.T) {
	want := map[string]Machine{
		macUpdateStage: OnMac, toolsStage: OnMac, homebrewStage: OnMac, answers: OnMac, macKey: OnMac,
		ghStage: OnMac, setupStage: OnMac, hostStage: OnMac, loginStage: OnMac, sudoStage: OnWs,
		installStage: OnWs, runStage: OnWs, claudeStage: OnWs, signingStage: OnMac, tunnelStage: OnMac,
		cableStage: OnMac, valuesStage: OnWs,
	}
	stages := []step.Step{&MacUpdate{}, &Tools{}, &Homebrew{}, &Answers{}, &MacKey{}, &Gh{}, &Setup{},
		&HostKey{}, &Login{}, &Sudo{}, &WsInstall{}, &WsRun{}, &WsClaude{}, &Signing{GitHub: &MacGitHub{}}, &Tunnel{}, &Cable{},
		&WsValues{}}
	if len(stages) != len(want) {
		t.Fatalf("%d stages, %d wanted", len(stages), len(want))
	}
	for _, s := range stages {
		got, ok := MachineOf(s)
		if !ok {
			t.Errorf("%s names no machine", s.Name())
			continue
		}
		if got != want[s.Name()] {
			t.Errorf("%s is on %s, want %s", s.Name(), got, want[s.Name()])
		}
	}
}

func TestMachineOfAStepWithNone(t *testing.T) {
	if m, ok := MachineOf(nil); ok || m != "" {
		t.Errorf("MachineOf(nil) = %q, %v", m, ok)
	}
}
