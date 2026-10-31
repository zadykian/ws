package setup_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/setup"
)

func ghosttyStep(m *effecttest.Machine, out *effecttest.Out) *setup.Ghostty {
	m.Files["/mac/ws.ghostty"] = []byte("font-size = 13\n")
	m.Dirs["/h/Applications/Ghostty.app"] = 0o755
	m.Files["/h/Library/Fonts/JetBrainsMonoNerdFontMono-Regular.ttf"] = nil
	return &setup.Ghostty{
		Home: "/h", Settings: "/mac/ws.ghostty", M: m, Out: out,
		LookPath: func(string) (string, error) { return "brew", nil },
		Real:     func(p string) (string, error) { return p, nil },
	}
}

func TestGhosttyAppAndFontThereNeedNoBrew(t *testing.T) {
	m := effecttest.NewMachine()
	p, err := ghosttyStep(m, &effecttest.Out{}).Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || slices.ContainsFunc(m.Calls, func(c string) bool {
		return strings.HasPrefix(c, "run ")
	}) {
		t.Errorf("Plan = %v, %v, calls %v; want none", p.Changes, err, m.Calls)
	}
}

func TestGhosttyFailedInstallIsAWarningAndStopsTheRun(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := ghosttyStep(m, out)
	delete(m.Dirs, "/h/Applications/Ghostty.app")
	m.Results["brew list --cask ghostty"] = effecttest.Result{Err: errors.New("no")}
	m.Results["brew list --cask font-jetbrains-mono-nerd-font"] = effecttest.Result{Err: errors.New("no")}
	m.Results["brew install --cask ghostty"] = effecttest.Result{Err: errors.New("down")}
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 1 {
		t.Fatalf("Plan = %v, %v; want the install", p.Changes, err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if p, err = s.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("second Plan = %v, %v; want none", p.Changes, err)
	}
}

func TestGhosttyWarnsWhereBrewHasTheCaskButNoApp(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := ghosttyStep(m, out)
	delete(m.Dirs, "/h/Applications/Ghostty.app")
	m.Results["brew list --cask ghostty"] = effecttest.Result{}
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || len(out.Lines) != 1 {
		t.Errorf("Plan = %v, %v, out %v; want one warning", p.Changes, err, out.Lines)
	}
}
