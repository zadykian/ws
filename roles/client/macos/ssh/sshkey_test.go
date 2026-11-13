package ssh_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

func TestSSHWritesTheCopyItNames(t *testing.T) {
	m, out := mactest.AgentMachine(sshMachine(), mactest.AgentKey), &effecttest.Out{}
	m.Results["mkdir -p /h/.config/ws/secretive"] = effecttest.Result{}
	s := sshStep(m, out)
	s.Conf.IdentityPubKey = sshcfg.AgentKeys(mactest.AgentKey)[0].Pub()
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if string(m.Files[mactest.CopyOf(mactest.AgentKey)]) != mactest.AgentKey {
		t.Errorf("the copy holds %q", m.Files[mactest.CopyOf(mactest.AgentKey)])
	}
	if got := string(m.Files["/h/.ssh/config.d/ws"]); !strings.Contains(got, "IdentityFile "+sshcfg.AgentKeys(mactest.AgentKey)[0].Pub()+"\n") {
		t.Errorf("config.d/ws names no copy: %s", got)
	}
	if again, err := s.Plan(context.Background()); err != nil || len(again.Changes) != 0 {
		t.Errorf("second Plan = %v, %v; want none", again.Changes, err)
	}
}

func TestSSHReportsTheDirectoryOfTheCopyItNames(t *testing.T) {
	m := mactest.AgentMachine(sshMachine(), mactest.AgentKey)
	s := sshStep(m, &effecttest.Out{})
	s.Conf.IdentityPubKey = sshcfg.AgentKeys(mactest.AgentKey)[0].Pub()
	got := mactest.ReportOf(t, s)
	for _, name := range []string{"/h/.config", "/h/.config/ws", "/h/" + sshcfg.KeyCopies} {
		if it := got[name]; it.Kind != step.DirItem || it.Found != "missing" {
			t.Errorf("%s = %+v, want a directory, missing", name, it)
		}
	}
}
