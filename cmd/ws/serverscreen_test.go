package main

import (
	"context"
	"errors"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/step"
)

const (
	// enterScreen and leaveScreen are how a terminal's alternate screen starts and ends.
	enterScreen = "\x1b[?1049h"
	leaveScreen = "\x1b[?1049l"
	// dpkgCheck is the look at a package that a run's Prepare makes first.
	dpkgCheck = "dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n ca-certificates"
)

// serverTerminal returns a process with a terminal, as root on the one system that ws server apply
// takes, with ca-certificates installed.
func serverTerminal() (process, *effecttest.Machine) {
	p := withEnv(fakeProcess(), map[string]string{"TERM": "xterm"})
	p.tty, p.errTTY, p.euid = true, true, 0
	m := p.machine.(*effecttest.Machine)
	m.Files["/etc/os-release"] = []byte("ID=ubuntu\nVERSION_ID=\"26.04\"\n")
	m.Results[dpkgCheck] = effecttest.Result{Out: []byte("ca-certificates ii \n")}
	return p, m
}

func TestServerApplyAtATerminalEndsOnTheNormalScreen(t *testing.T) {
	p, m := serverTerminal()
	m.Results["dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n tmux"] = effecttest.Result{Out: []byte("tmux ii \n")}
	status, stdout, stderr := runWs(p, "", "server", "apply", "--check", "--only", "tmux")
	left := strings.LastIndex(stderr, leaveScreen)
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || left < 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if want := "ws server apply: would change: none\n"; !strings.HasSuffix(stderr[left:], want) {
		t.Errorf("after the screen: %q, want the last line %q", stderr[left:], want)
	}
}

func TestServerApplyReportsAFailurePastTheScreen(t *testing.T) {
	p, m := serverTerminal()
	delete(m.Results, dpkgCheck)
	status, _, drawn := runWs(p, "", "server", "apply", "--only", "tmux")
	left := strings.LastIndex(drawn, leaveScreen)
	if status != 1 || left < 0 || !strings.Contains(drawn[left:], "ws server apply: Packages: ") {
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
func roleOf(name string, makes func(effect.Machine, effect.Out) step.Step) roles.Role {
	return roles.Role{Name: name, Steps: func(_ values.Values, _ secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
		return []step.Step{makes(m, out)}
	}}
}

func TestApplyOnScreenRunsTheRolesOnTheScreensEffects(t *testing.T) {
	p, m := serverTerminal()
	c, drawn := screenCommand(t, "")
	var made []effect.Out
	role := roleOf("a", func(_ effect.Machine, out effect.Out) step.Step {
		made = append(made, out)
		return stepOf{"one", func(context.Context) error { return nil }}
	})
	lines := &effecttest.Out{}
	cfg := apply.Config{Options: apply.Options{Check: true}, M: m, Out: lines, Observer: step.Printer{Out: lines}}
	changed, err := applyOnScreen(c, p, cfg, []roles.Role{role})
	if err != nil || !slices.Equal(changed, []string{"a"}) {
		t.Fatalf("changed = %q, err = %v", changed, err)
	}
	if !strings.Contains(drawn.String(), enterScreen) || !strings.Contains(drawn.String(), leaveScreen) {
		t.Errorf("no whole screen in %q", drawn.String())
	}
	if len(made) != 2 || made[0] != effect.Out(lines) {
		t.Fatalf("the steps were made on %v, want the plain effects first, for their names", made)
	}
	if _, ok := made[1].(*tui.Bridge); !ok {
		t.Errorf("the run's steps got %T, want the screen's bridge", made[1])
	}
	if len(lines.Lines) != 0 {
		t.Errorf("a run on the screen printed %q", lines.Lines)
	}
}

func TestApplyOnScreenStopsPastTheCurrentStep(t *testing.T) {
	p, m := serverTerminal()
	c, _ := screenCommand(t, "\x03\r")
	var reached bool
	waits := roleOf("a", func(_ effect.Machine, out effect.Out) step.Step {
		return stepOf{"one", func(ctx context.Context) error {
			stopped := out.(*tui.Bridge).Stop(ctx)
			for stopped() == nil {
				time.Sleep(time.Millisecond)
			}
			return nil
		}}
	})
	next := roleOf("b", func(effect.Machine, effect.Out) step.Step {
		return stepOf{"two", func(context.Context) error { reached = true; return nil }}
	})
	_, err := applyOnScreen(c, p, apply.Config{M: m, Out: &effecttest.Out{}}, []roles.Role{waits, next})
	if !errors.Is(err, signalExit{syscall.SIGINT}) || reached {
		t.Errorf("err = %v, the next role reached = %v", err, reached)
	}
}

func TestServerGroupsNameEveryStepOfARun(t *testing.T) {
	role := roleOf("a", func(effect.Machine, effect.Out) step.Step { return stepOf{name: "one"} })
	cfg := apply.Config{M: effecttest.NewMachine(), Prepare: []step.Step{stepOf{name: "first"}}}
	got := serverGroups(cfg, []roles.Role{role})
	want := []tui.Group{{Name: apply.Prepare, Steps: []string{"first"}}, {Name: "a", Steps: []string{"one"}}}
	if len(got) != 2 || got[0].Name != want[0].Name || !slices.Equal(got[0].Steps, want[0].Steps) ||
		got[1].Name != want[1].Name || !slices.Equal(got[1].Steps, want[1].Steps) {
		t.Errorf("groups = %+v, want %+v", got, want)
	}
}

// A role's steps follow the machine's values, so the check reads those first, and runs no step. A
// step's name holds a space, and --only keeps it whole.
func TestServerApplyRefusesAStepItsRoleLacks(t *testing.T) {
	p, m := serverTerminal()
	status, _, stderr := runWs(p, "", "server", "apply", "--only", "tmux/no such")
	want := `ws server apply: no step "no such" in tmux; its steps here are ["Tmux" "Tmux's bar"]` + "\n" +
		"Run 'ws server apply --help' for usage.\n"
	if status != 2 || stderr != want {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if want := []string{"read /etc/os-release", "read /etc/ws/host_vars.yml"}; !slices.Equal(m.Calls, want) {
		t.Errorf("calls = %q, want %q", m.Calls, want)
	}
}

func TestServerApplyTakesAStepByName(t *testing.T) {
	p, m := serverTerminal()
	m.Results["dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n tmux"] = effecttest.Result{Out: []byte("tmux ii \n")}
	status, _, stderr := runWs(p, "", "server", "apply", "--plain", "--check", "--only", "tmux/Tmux")
	if status != 0 || !strings.HasSuffix(stderr, "ws server apply: would change: none\n") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

func TestApplyOnScreenChecksTheStepsNamed(t *testing.T) {
	p, m := serverTerminal()
	c, drawn := screenCommand(t, "")
	role := roles.Role{Name: "a", Steps: func(values.Values, secrets.Secrets, effect.Machine, effect.Out) []step.Step {
		plan := func(context.Context) error { return nil }
		return []step.Step{stepOf{"one", plan}, stepOf{"two", plan}}
	}}
	cfg := apply.Config{Options: apply.Options{Check: true, Only: []string{"a/two"}}, M: m, Out: &effecttest.Out{}}
	changed, err := applyOnScreen(c, p, cfg, []roles.Role{role})
	if err != nil || !slices.Equal(changed, []string{"a"}) {
		t.Fatalf("changed = %q, err = %v", changed, err)
	}
	past := drawn.String()[strings.LastIndex(drawn.String(), leaveScreen):]
	if !strings.Contains(past, "a: two: x") || strings.Contains(past, "a: one: x") {
		t.Errorf("after the screen: %q, want the change of two alone", past)
	}
}

func TestServerApplyDrawsNothingWithPlain(t *testing.T) {
	p, m := serverTerminal()
	m.Results["dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n tmux"] = effecttest.Result{Out: []byte("tmux ii \n")}
	status, _, stderr := runWs(p, "", "server", "apply", "--plain", "--check", "--only", "tmux")
	if status != 0 || strings.Contains(stderr, "\x1b") || !strings.HasSuffix(stderr, "ws server apply: would change: none\n") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}
