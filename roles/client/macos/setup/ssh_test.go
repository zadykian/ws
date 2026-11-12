package setup_test

import (
	"context"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/roles/client/macos/setup"
)

// sshMachine answers the commands that the ssh step runs, with ssh happy, and has the file key that
// sshStep's mac.conf names.
func sshMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Files["/h/.ssh/id_ed25519.pub"] = []byte("ssh-ed25519 AAAA me\n")
	m.Results["ssh-keygen -l -f /h/.ssh/id_ed25519.pub"] = effecttest.Result{}
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/tmp/x\n")}
	m.Results["rm -rf /tmp/x"] = effecttest.Result{}
	for _, file := range []string{"new", "whole"} {
		m.Results["ssh -G -F /tmp/x/"+file+" ws"] = effecttest.Result{}
	}
	return m
}

func sshStep(m *effecttest.Machine, out *effecttest.Out) setup.SSH {
	return setup.SSH{
		Conf: conf.Conf{User: "me", IdentityPubKey: "~/.ssh/id_ed25519.pub"}, Home: "/h", M: m, Out: out,
		Now: func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) },
	}
}

func TestSSHPlanOfAFreshHome(t *testing.T) {
	m := sshMachine()
	p, err := sshStep(m, &effecttest.Out{}).Plan(context.Background())
	if err != nil || len(p.Changes) != 4 {
		t.Fatalf("Plan = %v, %v; want 4 changes", p.Changes, err)
	}
	if !strings.HasPrefix(p.Changes[2].String(), "--- /dev/null\n+++ /h/.ssh/config.d/ws\n") {
		t.Errorf("third change = %s, want config.d/ws's diff", p.Changes[2])
	}
}

func TestSSHApplyThenPlanIsEmpty(t *testing.T) {
	m, out := sshMachine(), &effecttest.Out{}
	s := sshStep(m, out)
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if again, err := s.Plan(context.Background()); err != nil || len(again.Changes) != 0 {
		t.Errorf("second Plan = %v, %v; want none", again.Changes, err)
	}
	if got := string(m.Files["/h/.ssh/config"]); got != "Include config.d/ws\n" {
		t.Errorf("~/.ssh/config = %q", got)
	}
}

func TestSSHRefusesAFileThatIsNotWss(t *testing.T) {
	m := sshMachine()
	m.Dirs["/h/.ssh"], m.Dirs["/h/.ssh/config.d"] = 0o700, 0o700
	m.Files["/h/.ssh/config.d/ws"] = []byte("Host mine\n")
	_, err := sshStep(m, &effecttest.Out{}).Plan(context.Background())
	if err == nil || !strings.Contains(err.Error(), "is not ws mac apply's") {
		t.Errorf("Plan err = %v, want a refusal of a file that is not ws's", err)
	}
}

func TestSSHRefusesAnIncludeBelowASetting(t *testing.T) {
	m := sshMachine()
	m.Files["/h/.ssh/config"] = []byte("Host a\nInclude config.d/ws\n")
	_, err := sshStep(m, &effecttest.Out{}).Plan(context.Background())
	if err == nil || !strings.Contains(err.Error(), "line 2 of /h/.ssh/config") {
		t.Errorf("Plan err = %v, want a refusal of the Include below a setting", err)
	}
}

func TestSSHWarnsOfBlocksForWs(t *testing.T) {
	m, out := sshMachine(), &effecttest.Out{}
	m.Files["/h/.ssh/config"] = []byte("Include config.d/ws\nHost ws\n  User x\n")
	if _, err := sshStep(m, out).Plan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(out.Lines) != 1 || !strings.HasPrefix(out.Lines[0], "warn: line 2 of ~/.ssh/config, Host ws, sets its own values for ws:") {
		t.Errorf("said %q, want a warning of line 2", out.Lines)
	}
}

func TestSSHWarnsOfABlockAtTheLineTheRunLeaves(t *testing.T) {
	m, out := sshMachine(), &effecttest.Out{}
	m.Files["/h/.ssh/config"] = []byte("Host ws\n  User x\n")
	m.Results["cp -p /h/.ssh/config /h/.ssh/config.20260102030405~"] = effecttest.Result{}
	if _, err := step.RunWith(t.Context(), []step.Step{sshStep(m, out)}, step.Options{}); err != nil {
		t.Fatal(err)
	}
	// The runner plans twice; Out's callers drop a text they have shown.
	warned := map[string]bool{}
	for _, l := range out.Lines {
		if strings.HasPrefix(l, "warn: ") {
			warned[l] = true
		}
	}
	want := "warn: line 3 of ~/.ssh/config, Host ws, sets its own values for ws:"
	if len(warned) != 1 || !strings.HasPrefix(slices.Collect(maps.Keys(warned))[0], want) {
		t.Errorf("warned %q, want one warning of line 3", slices.Collect(maps.Keys(warned)))
	}
}

func TestSSHAgainAppliesNothingOfAnSSHLeftOut(t *testing.T) {
	m, out := sshMachine(), &effecttest.Out{}
	again := sshStep(m, out)
	again.Again = true
	left := func(_, name string) bool { return name == "ssh" }
	if _, err := step.RunWith(t.Context(), []step.Step{sshStep(m, out), again}, step.Options{LeftOut: left}); err != nil {
		t.Fatal(err)
	}
	if i := slices.IndexFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "write /h/.ssh/") }); i >= 0 {
		t.Errorf("a left-out ssh applied: %s", m.Calls[i])
	}
}

func TestSSHStopsWhereItsDirectoryIsAFile(t *testing.T) {
	m := sshMachine()
	m.Files["/h/.ssh"] = []byte("x")
	if _, err := sshStep(m, &effecttest.Out{}).Plan(context.Background()); err == nil ||
		!strings.Contains(err.Error(), "/h/.ssh is not a directory") {
		t.Errorf("Plan's error = %v, want that /h/.ssh is no directory", err)
	}
}
