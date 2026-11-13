package cli

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strings"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// serverTerminal returns a process with a terminal, as root on the one system that ws server ubuntu
// apply takes, with ca-certificates and tmux installed.
func serverTerminal() (enginecli.Env, *effecttest.Machine) {
	p := withEnv(fakeProcess(), map[string]string{"TERM": "xterm"})
	p.TTY, p.ErrTTY, p.Euid = true, true, 0
	m := p.Machine.(*effecttest.Machine)
	m.Files["/etc/os-release"] = []byte("ID=ubuntu\nVERSION_ID=\"26.04\"\n")
	m.Packages = effecttest.Packages{"ca-certificates": {Version: "1", Status: "ii"}, "tmux": {Version: "3.5", Status: "ii"}}
	return p, m
}

func TestServerApplyAtATerminalEndsOnTheNormalScreen(t *testing.T) {
	p, _ := serverTerminal()
	status, stdout, stderr := runWs(p, "", "server", "ubuntu", "apply", "--check", "--only", "tmux")
	left := strings.LastIndex(stderr, leaveScreen)
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || left < 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if want := "ws server ubuntu apply: would change: none\n"; !strings.HasSuffix(stderr[left:], want) {
		t.Errorf("after the screen: %q, want the last line %q", stderr[left:], want)
	}
}

func TestServerApplyReportsAFailurePastTheScreen(t *testing.T) {
	p, m := serverTerminal()
	m.Packages = nil
	status, _, drawn := runWs(p, "", "server", "ubuntu", "apply", "--only", "tmux")
	left := strings.LastIndex(drawn, leaveScreen)
	if status != 1 || left < 0 || !strings.Contains(drawn[left:], "ws server ubuntu apply: Packages: ") {
		t.Errorf("status = %d, stderr = %q", status, drawn)
	}
}

// change is what a stepOf plans.
type change string

func (c change) String() string { return string(c) }

// stepOf is a step whose Plan runs plan and finds a change, so that --check names its role.
type stepOf struct {
	name string
	plan func(ctx context.Context) error
}

func (s stepOf) Name() string    { return s.name }
func (s stepOf) WaitsOn() string { return "" }
func (s stepOf) Plan(ctx context.Context) (step.Plan, error) {
	if err := s.plan(ctx); err != nil {
		return step.Plan{}, err
	}
	return step.Plan{Changes: []step.Change{change("x")}}, nil
}
func (stepOf) Apply(context.Context, step.Plan) error { return nil }

// roleOf is a role of one step, which makes takes the run's effects to build.
func roleOf(name string, makes func(effect.Machine, effect.Out) step.Step) role.Role {
	return role.Role{Name: name, Steps: func(m effect.Machine, _ effect.Units, out effect.Out) []step.Step {
		return []step.Step{makes(m, out)}
	}}
}

// A role's steps follow the machine's values, so the check reads those first, and runs no step. A
// step's name holds a space, and --only keeps it whole.
func TestServerApplyRefusesAStepItsRoleLacks(t *testing.T) {
	p, m := serverTerminal()
	status, _, stderr := runWs(p, "", "server", "ubuntu", "apply", "--only", "tmux/no such")
	want := `ws server ubuntu apply: no step "no such" in tmux; its steps here are ["Tmux"]` + "\n" +
		"Run 'ws server ubuntu apply --help' for usage.\n"
	if status != 2 || stderr != want {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if want := []string{"read /etc/os-release", "read /etc/ws/host_vars.yml"}; !slices.Equal(m.Calls, want) {
		t.Errorf("calls = %q, want %q", m.Calls, want)
	}
}

func TestServerApplyTakesAStepByName(t *testing.T) {
	p, _ := serverTerminal()
	status, _, stderr := runWs(p, "", "server", "ubuntu", "apply", "--plain", "--check", "--only", "tmux/Tmux")
	if status != 0 || !strings.HasSuffix(stderr, "ws server ubuntu apply: would change: none\n") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

func TestServerApplyDrawsNothingWithPlain(t *testing.T) {
	p, _ := serverTerminal()
	status, _, stderr := runWs(p, "", "server", "ubuntu", "apply", "--plain", "--check", "--only", "tmux")
	if status != 0 || strings.Contains(stderr, "\x1b") || !strings.HasSuffix(stderr, "ws server ubuntu apply: would change: none\n") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

func TestServerApplyOpensTheListAndLeavesWithQ(t *testing.T) {
	p, _ := serverTerminal()
	status, stdout, stderr := runWs(p, "q", "server", "ubuntu", "apply")
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if strings.Contains(stderr, "changed:") {
		t.Errorf("a run began: %q", stderr)
	}
}

func TestServerApplyChecksFromTheListAndSaysSo(t *testing.T) {
	p, m := serverTerminal()
	// The list scans every role, and base would install what the table leaves out.
	for _, name := range values.Default().Base.Packages {
		m.Packages[name] = effecttest.Package{Version: "1", Status: "ii"}
	}
	var errs bytes.Buffer
	root := enginecli.Root(p, serverGroup(p))
	// A scan holds c back, so each turn ends by emptying the filter that the next one types.
	status := enginecli.Execute(t.Context(), root, []string{"server", "ubuntu", "apply"}, repeating("/tmux\rac/\x7f\x7f\x7f\x7f\r"), io.Discard, &errs)
	if want := "ws server ubuntu apply: would change: none\n"; status != 0 || !strings.HasSuffix(errs.String(), want) {
		t.Errorf("status = %d, stderr ends %q, want %q", status, errs.String()[max(0, errs.Len()-300):], want)
	}
}
