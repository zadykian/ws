package zed_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/zed"
)

const zedKeymap = "// Written by github.com/zadykian/ws, whose next run replaces any change.\n[]\n"

func zedStep(m *effecttest.Machine, out *effecttest.Out) *zed.Zed {
	m.Dirs["/h/Applications/Zed.app"] = 0o755
	m.Files["/h/Library/Fonts/JetBrainsMonoNerdFontMono-Regular.ttf"] = nil
	return &zed.Zed{
		Home: "/h", Settings: []byte("// Written by ws client macos apply: its next run replaces changes made here.\n{}\n"),
		Keymap: []byte(zedKeymap), M: m, Out: out,
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
	if err != nil || len(p.Changes) != 6 {
		t.Fatalf("Plan = %v, %v; want the install and the files", p.Changes, err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	_, wrote := m.Files["/h/.config/zed/global_settings.json"]
	if wrote || !strings.Contains(strings.Join(out.Lines, "\n"), "brew install --cask zed failed") {
		t.Errorf("out %v, wrote Zed's files %v; want a warning and no files", out.Lines, wrote)
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
	if err != nil || len(p.Changes) != 5 || !strings.Contains(p.Changes[0].String(), "/h/.config,") {
		t.Fatalf("Plan = %v, %v; want ~/.config, ~/.config/zed and the three files", p.Changes, err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	want := "// Written by ws client macos apply: its next run replaces changes made here.\n{}\n"
	if got := string(m.Files["/h/.config/zed/global_settings.json"]); got != want {
		t.Errorf("global_settings.json = %q", got)
	}
	if got := string(m.Files["/h/.config/zed/keymap.json"]); got != zedKeymap {
		t.Errorf("keymap.json = %q; want the keymap as it is", got)
	}
	if p, err = s.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("second Plan = %v, %v; want none", p.Changes, err)
	}
}

func TestZedSeedIsJSONWithItsCommentsDropped(t *testing.T) {
	m := effecttest.NewMachine()
	s := zedStep(m, &effecttest.Out{})
	p, _ := s.Plan(context.Background())
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, l := range strings.Split(string(m.Files["/h/.config/zed/settings.json"]), "\n") {
		if !strings.HasPrefix(l, "//") {
			lines = append(lines, l)
		}
	}
	var seed struct {
		Connections []struct{ Host string } `json:"ssh_connections"`
	}
	if err := json.Unmarshal([]byte(strings.Join(lines, "\n")), &seed); err != nil ||
		len(seed.Connections) != 1 || seed.Connections[0].Host != "ws" {
		t.Errorf("seed = %+v, %v; want the host ws", seed, err)
	}
}

func TestZedLeavesAnExistingSettingsFileAlone(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := zedStep(m, out)
	m.Files["/h/.config/zed/settings.json"] = []byte("{ // mine\n}\n")
	p, _ := s.Plan(context.Background())
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/h/.config/zed/settings.json"]); got != "{ // mine\n}\n" || strings.Contains(strings.Join(out.Lines, "\n"), "warn") {
		t.Errorf("settings.json = %q, out %v; want it left, with no warning", got, out.Lines)
	}
}

func TestZedWarnsOfAFileNotItsOwn(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := zedStep(m, out)
	m.Files["/h/.config/zed/global_settings.json"] = []byte("{}\n")
	p, err := s.Plan(context.Background())
	if err != nil || !strings.Contains(strings.Join(out.Lines, "\n"), "is not ws's: move it away") {
		t.Errorf("Plan = %v, %v, out %v; want a warning", p.Changes, err, out.Lines)
	}
	for _, c := range p.Changes {
		if strings.Contains(c.String(), "/zed/global_settings.json") {
			t.Errorf("Plan = %v; want the file left", p.Changes)
		}
	}
}

func TestZedWarnsOfALinkedDirectoryWhereAWriteIsPlanned(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := zedStep(m, out)
	m.Dirs["/dots/zed"] = 0o755
	m.Links["/h/.config/zed"] = "/dots/zed"
	if _, err := s.Plan(context.Background()); err != nil || len(out.Lines) != 1 ||
		!strings.Contains(out.Lines[0], "is a link to /dots/zed: Zed's files go there") {
		t.Errorf("out %v; want the link's warning", out.Lines)
	}
}

func TestZedReplacesAKeymapThatStartsWithTheMark(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := zedStep(m, out)
	m.Dirs["/h/.config/zed"] = 0o755
	old := "// Written by github.com/zadykian/ws, an older line 1.\n[]\n"
	m.Files["/h/.config/zed/keymap.json"] = []byte(old)
	p, _ := s.Plan(context.Background())
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/h/.config/zed/keymap.json"]); got != zedKeymap || strings.Contains(strings.Join(out.Lines, "\n"), "warn") {
		t.Errorf("keymap.json = %q, out %v; want ws's, with no warning", got, out.Lines)
	}
	if p, err := s.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("second Plan = %v, %v; want none", p.Changes, err)
	}
}

func TestZedLeavesAKeymapWithAnotherFirstLineAndWarnsOnce(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := zedStep(m, out)
	m.Dirs["/h/.config/zed"] = 0o755
	m.Files["/h/.config/zed/keymap.json"] = []byte("[]\n")
	p, _ := s.Plan(context.Background())
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/h/.config/zed/keymap.json"]); got != "[]\n" ||
		strings.Count(strings.Join(out.Lines, "\n"), "keymap.json is not ws's: move it away") != 1 {
		t.Errorf("keymap.json = %q, out %v; want it left, with one warning", got, out.Lines)
	}
}
