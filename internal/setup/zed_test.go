package setup_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/setup"
)

func zedStep(m *effecttest.Machine, out *effecttest.Out) *setup.Zed {
	m.Dirs["/h/Applications/Zed.app"] = 0o755
	m.Files["/h/Library/Fonts/JetBrainsMonoNerdFontMono-Regular.ttf"] = nil
	return &setup.Zed{
		Home: "/h", Settings: []byte("{}\n"), M: m, Out: out,
		LookPath: func(string) (string, error) { return "brew", nil },
	}
}

func TestZedAppAndFontThereNeedNoBrew(t *testing.T) {
	m := effecttest.NewMachine()
	p, err := zedStep(m, &effecttest.Out{}).Plan(context.Background())
	if err != nil || slices.ContainsFunc(m.Calls, func(c string) bool {
		return strings.HasPrefix(c, "run ")
	}) {
		t.Errorf("Plan = %v, %v, calls %v; want no brew", p.Changes, err, m.Calls)
	}
}

func TestZedFailedInstallIsAWarningAndTheSecondPlanIsEmpty(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := zedStep(m, out)
	delete(m.Dirs, "/h/Applications/Zed.app")
	m.Results["brew list --cask zed"] = effecttest.Result{Err: errors.New("no")}
	m.Results["brew install --cask zed"] = effecttest.Result{Err: errors.New("down")}
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 4 {
		t.Fatalf("Plan = %v, %v; want the install and the files", p.Changes, err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(out.Lines, "\n"), "brew install --cask zed failed") {
		t.Errorf("out %v; want a warning", out.Lines)
	}
	if p, err = s.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("second Plan = %v, %v; want none", p.Changes, err)
	}
}

func TestZedWarnsOnceWhereBrewHasTheCaskButNoApp(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := zedStep(m, out)
	delete(m.Dirs, "/h/Applications/Zed.app")
	m.Results["brew list --cask zed"] = effecttest.Result{}
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || len(out.Lines) != 1 ||
		!strings.Contains(out.Lines[0], "brew reinstall --cask zed") {
		t.Errorf("Plan = %v, %v, out %v; want one warning", p.Changes, err, out.Lines)
	}
}

func TestZedFirstRunMakesTheDirectoriesAndWritesTheFile(t *testing.T) {
	m := effecttest.NewMachine()
	s := zedStep(m, &effecttest.Out{})
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 3 || !strings.Contains(p.Changes[0].String(), "/h/.config,") {
		t.Fatalf("Plan = %v, %v; want ~/.config, ~/.config/zed and the file", p.Changes, err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	want := "// Written by ws mac apply: its next run replaces changes made here.\n{}\n"
	if got := string(m.Files["/h/.config/zed/global_settings.json"]); got != want {
		t.Errorf("global_settings.json = %q", got)
	}
	if p, err = s.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("second Plan = %v, %v; want none", p.Changes, err)
	}
}
