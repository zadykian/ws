package ghostty_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/roles/client/macos/ghostty"
)

const ghosttyDir = "/h/Library/Application Support/com.mitchellh.ghostty"

var ghosttyHead = file.Header("#") + "\n"

func ghosttyStep(m *effecttest.Machine, out *effecttest.Out) resource.Lazy {
	m.Dirs["/h/Applications/Ghostty.app"] = 0o755
	m.Files["/h/Library/Fonts/JetBrainsMonoNerdFontMono-Regular.ttf"] = nil
	return (&ghostty.Ghostty{
		Home: "/h", Settings: []byte("font-size = 13\n"), M: m, Out: out,
		LookPath: func(string) (string, error) { return "brew", nil },
		Real:     func(p string) (string, error) { return p, nil },
		Now:      func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) },
	}).Step()
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
	delete(m.Files, "/h/Library/Fonts/JetBrainsMonoNerdFontMono-Regular.ttf")
	m.Results["brew list --cask ghostty"] = effecttest.Result{Err: errors.New("no")}
	m.Results["brew list --cask font-jetbrains-mono-nerd-font"] = effecttest.Result{Err: errors.New("no")}
	m.Results["brew install --cask ghostty"] = effecttest.Result{Err: errors.New("down")}
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 6 {
		t.Fatalf("Plan = %v, %v; want the installs and the files", p.Changes, err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files["/etc/ws/ghostty"]; ok || !slices.Contains(out.Lines, "warn: brew install --cask ghostty failed") {
		t.Errorf("/etc/ws/ghostty written %v, out %q; want a warning and the files held", ok, out.Lines)
	}
	if !slices.Contains(m.Calls, "run brew install --cask font-jetbrains-mono-nerd-font") {
		t.Errorf("calls %v; want the font's cask installed after the app's failure", m.Calls)
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
	if err != nil || len(p.Changes) != 4 {
		t.Fatalf("Plan = %v, %v; want /etc/ws, its file, a directory and the config", p.Changes, err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/etc/ws/ghostty"]); got != ghosttyHead+"font-size = 13\n" {
		t.Errorf("/etc/ws/ghostty = %q", got)
	}
	if got := string(m.Files[ghosttyDir+"/config.ghostty"]); got != "config-file = ?/etc/ws/ghostty\n" {
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
	m.Results["cp -p /h/.config/ghostty/config /h/.config/ghostty/config.20260102030405~"] = effecttest.Result{}
	p, _ := s.Plan(context.Background())
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/h/.config/ghostty/config"]); got != "theme = x\nconfig-file = ?/etc/ws/ghostty\n" {
		t.Errorf("config = %q", got)
	}
	if m.Modes["/h/.config/ghostty/config"] != 0o600 {
		t.Errorf("mode = %o", m.Modes["/h/.config/ghostty/config"])
	}
}

// The file lies in /etc/ws, ws's by its place: a run replaces what lies there, and warns of nothing.
func TestGhosttyReplacesAFileInRootsDirectory(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := ghosttyStep(m, out)
	m.Dirs["/etc/ws"] = 0o755
	m.Files["/etc/ws/ghostty"] = []byte("mine\n")
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 3 || len(out.Lines) != 0 {
		t.Fatalf("Plan = %v, %v, out %v; want the file, a directory and the config, no warning", p.Changes, err, out.Lines)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/etc/ws/ghostty"]); got != ghosttyHead+"font-size = 13\n" {
		t.Errorf("/etc/ws/ghostty = %q", got)
	}
}

// A second plan asks for no password: root's file and directory read as the user.
func TestGhosttyAsksNoRootWhereNothingChanges(t *testing.T) {
	m := effecttest.NewMachine()
	s := ghosttyStep(m, &effecttest.Out{})
	p, _ := s.Plan(context.Background())
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	m.Calls = nil
	if p, err := s.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Fatalf("second Plan = %v, %v; want none", p.Changes, err)
	}
	if i := slices.IndexFunc(m.Calls, func(c string) bool { return strings.Contains(c, "root ") }); i >= 0 {
		t.Errorf("the second plan asked for root: %q", m.Calls[i])
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
	if string(m.Files[ghosttyDir+"/config.ghostty"]) != "a = 1\n" || !strings.Contains(strings.Join(out.Lines, "\n"), "add 'config-file = ?/etc/ws/ghostty'") {
		t.Errorf("config %q, out %v", m.Files[ghosttyDir+"/config.ghostty"], out.Lines)
	}
}

// firstRun is ghosttyStep on a Mac whose /etc/ws is missing, as before ws's first run: it does not
// resolve, as filepath.EvalSymlinks fails on a missing directory.
func firstRun(m *effecttest.Machine, out *effecttest.Out) resource.Lazy {
	m.Dirs["/h/Applications/Ghostty.app"] = 0o755
	m.Files["/h/Library/Fonts/JetBrainsMonoNerdFontMono-Regular.ttf"] = nil
	return (&ghostty.Ghostty{
		Home: "/h", Settings: []byte("font-size = 13\n"), M: m, Out: out,
		LookPath: func(string) (string, error) { return "brew", nil },
		Real: func(p string) (string, error) {
			if p == "/etc/ws" {
				return "", errors.New("no such directory")
			}
			return p, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) },
	}).Step()
}

func TestGhosttyFirstRunKeepsTheIncludeLineThatTheConfigHolds(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := firstRun(m, out)
	config := ghosttyDir + "/config.ghostty"
	m.Files[config] = []byte("a = 1\nconfig-file = ?/etc/ws/ghostty\n")
	p, err := s.Plan(context.Background())
	if err == nil {
		err = s.Apply(context.Background(), p)
	}
	if err != nil {
		t.Fatal(err)
	}
	if string(m.Files[config]) != "a = 1\nconfig-file = ?/etc/ws/ghostty\n" {
		t.Errorf("config %q, want it unchanged", m.Files[config])
	}
	for name := range m.Files {
		if strings.HasPrefix(name, config+".") {
			t.Errorf("a backup %s of a config that needed none", name)
		}
	}
	if _, ok := m.Files["/etc/ws/ghostty"]; !ok {
		t.Error("the first run wrote no /etc/ws/ghostty")
	}
}

func TestGhosttyFirstRunDoesNotWarnOfALinkedConfigThatHoldsTheLine(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := firstRun(m, out)
	config := ghosttyDir + "/config.ghostty"
	m.Files[config] = []byte("config-file = ?/etc/ws/ghostty\n")
	m.Links[config] = "/dots/config"
	p, err := s.Plan(context.Background())
	if err == nil {
		err = s.Apply(context.Background(), p)
	}
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(out.Lines, "\n"); strings.Contains(got, "add 'config-file") {
		t.Errorf("a linked config that holds the line drew a warning: %v", out.Lines)
	}
}

func TestGhosttyStopsWhereItsDirectoryIsAFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[ghosttyDir] = []byte("x")
	if _, err := ghosttyStep(m, &effecttest.Out{}).Plan(context.Background()); err == nil ||
		!strings.Contains(err.Error(), "is not a directory") {
		t.Errorf("Plan's error = %v, want that the directory is no directory", err)
	}
}
