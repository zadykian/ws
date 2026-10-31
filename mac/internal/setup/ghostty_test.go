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

const (
	ghosttyDir  = "/h/Library/Application Support/com.mitchellh.ghostty"
	ghosttyHead = "# Written by ws's mac/setup.sh: its next run replaces changes made here.\n"
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
	if err != nil || slices.ContainsFunc(m.Calls, func(c string) bool {
		return strings.HasPrefix(c, "run ")
	}) {
		t.Errorf("Plan = %v, %v, calls %v; want no brew", p.Changes, err, m.Calls)
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
	if err != nil || len(p.Changes) != 4 {
		t.Fatalf("Plan = %v, %v; want the install and the files", p.Changes, err)
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

func TestGhosttyWritesItsFileAndTheLine(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := ghosttyStep(m, out)
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 3 {
		t.Fatalf("Plan = %v, %v; want a directory and two files", p.Changes, err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[ghosttyDir+"/ws.ghostty"]); got != ghosttyHead+"font-size = 13\n" {
		t.Errorf("ws.ghostty = %q", got)
	}
	if got := string(m.Files[ghosttyDir+"/config.ghostty"]); got != "config-file = ?ws.ghostty\n" {
		t.Errorf("config = %q", got)
	}
	if p, err = s.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("second Plan = %v, %v; want none", p.Changes, err)
	}
}

func TestGhosttyKeepsTheConfigsTextAndMode(t *testing.T) {
	m := effecttest.NewMachine()
	s := ghosttyStep(m, &effecttest.Out{})
	m.Files["/h/.config/ghostty/config"] = []byte("theme = x")
	m.Modes["/h/.config/ghostty/config"] = 0o600
	p, _ := s.Plan(context.Background())
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/h/.config/ghostty/config"]); got != "theme = x\nconfig-file = ?ws.ghostty\n" {
		t.Errorf("config = %q", got)
	}
	if m.Modes["/h/.config/ghostty/config"] != 0o600 {
		t.Errorf("mode = %o", m.Modes["/h/.config/ghostty/config"])
	}
}

func TestGhosttyWarnsOfAFileNotItsOwn(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := ghosttyStep(m, out)
	m.Files[ghosttyDir+"/ws.ghostty"] = []byte("mine\n")
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || !strings.Contains(strings.Join(out.Lines, "\n"), "is not mac/setup.sh's") {
		t.Errorf("Plan = %v, %v, out %v; want a warning alone", p.Changes, err, out.Lines)
	}
}

func TestGhosttyWarnsOfALinkedConfig(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := ghosttyStep(m, out)
	m.Files[ghosttyDir+"/config.ghostty"] = []byte("a = 1\n")
	m.Links[ghosttyDir+"/config.ghostty"] = "/dots/config"
	p, _ := s.Plan(context.Background())
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if string(m.Files[ghosttyDir+"/config.ghostty"]) != "a = 1\n" || !strings.Contains(strings.Join(out.Lines, "\n"), "add 'config-file = ?ws.ghostty'") {
		t.Errorf("config %q, out %v", m.Files[ghosttyDir+"/config.ghostty"], out.Lines)
	}
}
