package cli

import (
	"context"
	"errors"
	"io"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos"
	"github.com/zadykian/ws/roles/client/macos/stages/tools"
)

// lastEvents keeps the steps that end, each after its parent.
type lastEvents []string

func (l *lastEvents) Observe(e step.Event) {
	if e.Kind >= step.Done {
		*l = append(*l, e.Parent+"/"+e.Step)
	}
}

func TestWizardSetupPassesTheObserverOn(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"], m.Files["/h/servers/ws/server.conf"] = []byte(""), []byte("WS_USER=ada\n")
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = m
	var got lastEvents
	s := wizardSetup(wizardEffects{p: p, text: io.Discard, out: &effecttest.Out{}, obs: &got}, nil, "/h/mac.conf", "ws")
	// The fake machine fails ssh's plan, and each step ends all the same.
	_, _ = s.Run(context.Background(), true)
	var want lastEvents
	for _, name := range macos.Names() {
		want = append(want, s.Name()+"/"+name)
	}
	if !slices.Equal(got, want) {
		t.Errorf("steps = %q, want %q", got, want)
	}
}

// leftEvents keeps the steps that a run left out, and those that ended unreached.
type leftEvents struct{ left, skipped []string }

func (l *leftEvents) Observe(e step.Event) {
	switch {
	case e.LeftOut && e.Kind >= step.Done:
		l.left = append(l.left, e.Step)
	case e.Kind == step.Skipped && e.After == "" && e.Err == nil:
		l.skipped = append(l.skipped, e.Step)
	}
}

func setupOf(t *testing.T, e wizardEffects, left func(parent, name string) bool) *tools.Setup {
	t.Helper()
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"], m.Files["/h/servers/ws/server.conf"] = []byte(""), []byte("WS_USER=ada\n")
	e.p = withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	e.p.Machine, e.text, e.out = m, io.Discard, &effecttest.Out{}
	return wizardSetup(e, left, "/h/mac.conf", "ws")
}

func TestWizardSetupWaitsOnTheMacKeyStage(t *testing.T) {
	// setupOf's mac.conf names no key for ws, which the Mac key stage would ask for.
	s := setupOf(t, wizardEffects{}, nil)
	s.LookPath = func(string) (string, error) { return "/brew", nil }
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 1 || p.Changes[0].String() != "ws client macos apply: after Mac key" {
		t.Errorf("Plan = %v, %v; want the wait on the Mac key stage", p, err)
	}
}

func TestWizardSetupLeavesOutWhatTheRunDoes(t *testing.T) {
	var got leftEvents
	left := func(parent, name string) bool { return parent == "ws client macos apply" && name == "ssh" }
	s := setupOf(t, wizardEffects{obs: &got}, left)
	// The fake machine fails ssh's plan, which a step left out survives.
	_, _ = s.Run(context.Background(), true)
	if want := []string{"ssh"}; !slices.Equal(got.left, want) {
		t.Errorf("left out = %q, want %q", got.left, want)
	}
}

func TestWizardSetupStopsWhereTheRunDoes(t *testing.T) {
	var got leftEvents
	stopped := errors.New("stopped")
	s := setupOf(t, wizardEffects{obs: &got, stop: func() error { return stopped }}, nil)
	if _, err := s.Run(context.Background(), true); !errors.Is(err, stopped) {
		t.Errorf("err = %v, want the stop", err)
	}
	if len(got.skipped) != len(macos.Names()) {
		t.Errorf("unreached = %q; want every step", got.skipped)
	}
}

// TestWizardSetupListsTheServersBesideItsFile runs ssh's check for lab, new beside ws at slot 0.
// lab's own file names no slot yet, and Answers' would-be one gives it slot 1. The steps load ws
// beside that one, as the run will, and so find no clash.
func TestWizardSetupListsTheServersBesideItsFile(t *testing.T) {
	p, m := macTerminal()
	m.Files["/h/.config/ws/servers/lab/server.conf"], m.Dirs["/h/.config/ws/servers/lab"] = []byte("WS_USER=ada\n"), 0o755
	m.Files["/lock/mac.conf"] = m.Files["/h/.config/ws/mac.conf"]
	m.Files["/lock/servers/ws/server.conf"] = []byte("WS_USER=ada\n")
	m.Files["/lock/servers/lab/server.conf"] = []byte("WS_USER=ada\nWS_SLOT=1\n")
	m.Dirs["/lock/servers"], m.Dirs["/lock/servers/ws"], m.Dirs["/lock/servers/lab"] = 0o755, 0o755, 0o755
	for _, file := range []string{"new", "whole"} {
		m.Results["ssh -G -F /tmp/x/"+file+" lab"] = effecttest.Result{}
	}
	left := func(parent, name string) bool { return parent == "ws client macos apply" && name != "ssh" }
	s := wizardSetup(wizardEffects{p: p, text: io.Discard, out: &effecttest.Out{}}, left, "/lock/mac.conf", "lab")
	if _, err := s.Run(context.Background(), true); err != nil {
		t.Errorf("Run = %v, want the plan of the servers beside /lock/mac.conf", err)
	}
}
