package setup_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/setup"
)

// sshMachine answers the commands that the ssh step runs, with ssh happy.
func sshMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/tmp/x\n")}
	m.Results["rm -rf /tmp/x"] = effecttest.Result{}
	for _, file := range []string{"new", "whole"} {
		for _, host := range []string{"ws", "ws-ssh"} {
			m.Results["ssh -G -F /tmp/x/"+file+" "+host] = effecttest.Result{}
		}
	}
	return m
}

func sshStep(m *effecttest.Machine, out *effecttest.Out) setup.SSH {
	return setup.SSH{
		Conf: conf.Conf{User: "me"}, Home: "/h", M: m, Out: out,
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
	if err == nil || !strings.Contains(err.Error(), "is not mac/setup.sh's") {
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
