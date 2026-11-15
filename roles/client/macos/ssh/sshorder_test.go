package ssh_test

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// copyingSSH is a fresh Mac whose key is an agent's, so that ssh writes a copy of it.
func copyingSSH(out *effecttest.Out) step.Step {
	m := mactest.AgentMachine(sshMachine(), mactest.AgentKey)
	c := sshConf(m, out)
	c.Conf.IdentityPubKey = sshcfg.AgentKeys(mactest.AgentKey)[0].Pub()
	return c.Step()
}

// A plan and a run say the copy first, then the directories, config.d/ws and the Include. A report
// lists the directories, the two files, then the copy's.
func TestSSHPlansAndAppliesInOrder(t *testing.T) {
	out := &effecttest.Out{}
	s := copyingSSH(out)
	copied := mactest.CopyOf(mactest.AgentKey)
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var heads []string
	for _, c := range p.Changes {
		heads = append(heads, step.Headline(c))
	}
	wantHeads := []string{
		"create directory /etc/ws, owner 0:0, mode 755",
		"create directory /etc/ws/secretive, owner 0:0, mode 755",
		"create " + copied,
		"would make /h/.ssh, mode 700", "would make /h/.ssh/config.d, mode 700",
		"would create /h/.ssh/config.d/ws", "would create /h/.ssh/config",
	}
	if !slices.Equal(heads, wantHeads) {
		t.Fatalf("headlines %q; want %q", heads, wantHeads)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	said := []string{"say: create directory /etc/ws, owner 0:0, mode 755",
		"say: create directory /etc/ws/secretive, owner 0:0, mode 755", "say: create " + copied,
		"say: wrote /h/.ssh/config.d/ws", "say: wrote /h/.ssh/config"}
	if !slices.Equal(out.Lines, said) {
		t.Errorf("said %q; want %q", out.Lines, said)
	}
}

func TestSSHReportsItsFilesInOrder(t *testing.T) {
	items, err := copyingSSH(&effecttest.Out{}).(step.Reporter).Report(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, it := range items {
		names = append(names, it.Name)
	}
	want := []string{"/h/.ssh", "/h/.ssh/config.d", "/h/.ssh/config.d/ws", "/h/.ssh/config",
		"/etc/ws", "/etc/ws/secretive", mactest.CopyOf(mactest.AgentKey)}
	if !slices.Equal(names, want) {
		t.Errorf("items %q; want %q", names, want)
	}
}

// A run that finds config.d/ws with the wrong mode plans the mode alone.
func TestSSHPlansTheModeOfItsFileAlone(t *testing.T) {
	m := sshMachine()
	s := sshStep(m, &effecttest.Out{})
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	m.Modes["/h/.ssh/config.d/ws"] = 0o644
	again, err := s.Plan(context.Background())
	if err != nil || len(again.Changes) != 1 || step.Headline(again.Changes[0]) != "would set /h/.ssh/config.d/ws's mode to 600" {
		t.Errorf("Plan = %v, %v; want the mode alone", again.Changes, err)
	}
}
