package wizard

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/step"
)

func newTools(dir string, git bool) (*Tools, *effecttest.Machine, *effecttest.Out) {
	m := effecttest.NewMachine()
	out := &effecttest.Out{}
	if dir == "" {
		m.Results["xcode-select -p"] = effecttest.Result{Err: errors.New("exit status 2")}
	} else {
		m.Results["xcode-select -p"] = effecttest.Result{Out: []byte(dir + "\n")}
	}
	if git {
		m.Files[dir+"/usr/bin/git"] = nil
		m.Modes[dir+"/usr/bin/git"] = 0o755
	}
	m.Results["xcode-select --install"] = effecttest.Result{Err: errors.New("already open")}
	return &Tools{M: m, Out: out, Sleep: func(time.Duration) {}}, m, out
}

func TestToolsPlan(t *testing.T) {
	for name, tc := range map[string]struct {
		dir     string
		git     bool
		mode    fs.FileMode
		changes int
	}{
		"there":      {"/clt", true, 0o755, 0},
		"no dir":     {"", false, 0, 1},
		"no git":     {"/clt", false, 0, 1},
		"not a tool": {"/clt", true, 0o644, 1},
	} {
		tl, m, _ := newTools(tc.dir, tc.git)
		if tc.git {
			m.Modes["/clt/usr/bin/git"] = tc.mode
		}
		p, err := tl.Plan(context.Background())
		if err != nil || len(p.Changes) != tc.changes {
			t.Errorf("%s: Plan = %v, %v; want %d changes", name, p.Changes, err, tc.changes)
		}
		if slices.Contains(m.Calls, "run git --version") {
			t.Errorf("%s: ran git itself, whose shim opens the install", name)
		}
	}
}

func TestToolsApplyWaitsForTheDialog(t *testing.T) {
	tl, m, out := newTools("/clt", false)
	sleeps := 0
	tl.Sleep = func(d time.Duration) {
		if d != 10*time.Second {
			t.Errorf("slept %v, want 10s", d)
		}
		sleeps++
		if sleeps == 3 {
			m.Files["/clt/usr/bin/git"] = nil
			m.Modes["/clt/usr/bin/git"] = 0o755
		}
	}
	if err := tl.Apply(context.Background(), step.Plan{}); err != nil || sleeps != 3 {
		t.Fatalf("Apply = %v after %d sleeps, want nil after 3", err, sleeps)
	}
	if !slices.Contains(m.Calls, "run xcode-select --install") || len(out.Lines) == 0 ||
		!strings.Contains(strings.Join(out.Lines, "\n"), "waiting up to 60 minutes") {
		t.Errorf("calls = %v, out = %v", m.Calls, out.Lines)
	}
}

func TestToolsApplyGivesUpAfterAnHour(t *testing.T) {
	tl, _, _ := newTools("/clt", false)
	sleeps := 0
	tl.Sleep = func(time.Duration) { sleeps++ }
	err := tl.Apply(context.Background(), step.Plan{})
	if err == nil || !strings.Contains(err.Error(), "no Command Line Tools after 60 minutes") || sleeps != 360 {
		t.Errorf("Apply = %v after %d sleeps, want the failure after 360", err, sleeps)
	}
}
