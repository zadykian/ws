package zsh_test

import (
	"context"
	"github.com/zadykian/ws/engine/core/resource/file"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/zsh"
)

const zshSource = "source ~/.config/ws/zshrc\n"

func zshStep(m *effecttest.Machine, out *effecttest.Out) resource.Lazy {
	return (&zsh.Zsh{Home: "/h", Hook: zsh.Hook, M: m, Out: out,
		Now: func() time.Time { return time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC) }}).Step()
}

// The second plan finds roles/client/macos/zsh/files/zshrc's header, so the hook that ws embeds is its own.
func TestZshMakesTheHookAndZshrcThenNothing(t *testing.T) {
	m := effecttest.NewMachine()
	out := &effecttest.Out{}
	s := zshStep(m, out)
	p, err := s.Plan(context.Background())
	want := []step.Change{file.UserMkdir{M: m, Path: "/h/.config", Mode: 0o755}, file.UserMkdir{M: m, Path: "/h/.config/ws", Mode: 0o700},
		file.UserWrite{M: m, Out: out, Path: "/h/.config/ws/zshrc", After: zsh.Hook, Mode: 0o644},
		file.UserWrite{M: m, Out: out, Path: "/h/.zshrc", After: []byte(zshSource), Mode: 0o644}}
	if err != nil || !reflect.DeepEqual(p.Changes, want) {
		t.Fatalf("Plan = %v, %v; want %v", p.Changes, err, want)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if p, err := s.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("second Plan = %v, %v; want none", p.Changes, err)
	}
	items, err := s.Report(context.Background())
	if err != nil || len(items) != 4 || items[2].Conflict != "" || items[3].Found != items[3].Want {
		t.Errorf("Report = %+v, %v; want the directories, the hook and the line, no conflict", items, err)
	}
}

func TestZshLeavesAHookNotItsOwnAndZshrc(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Dirs["/h/.config"], m.Dirs["/h/.config/ws"] = 0o755, 0o700
	m.Files["/h/.config/ws/zshrc"] = []byte("echo mine\n")
	p, err := zshStep(m, out).Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || !strings.Contains(strings.Join(out.Lines, "\n"), "is not ws's") {
		t.Errorf("Plan = %v, %v, out %v; want nothing and a warning", p.Changes, err, out.Lines)
	}
}

func TestZshWarnsOfALinkedZshrcThatLacksTheLine(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Dirs["/h/.config"], m.Dirs["/h/.config/ws"] = 0o755, 0o700
	m.Files["/h/.config/ws/zshrc"] = zsh.Hook
	m.Files["/h/.zshrc"] = []byte("export EDITOR=vi\n")
	m.Links["/h/.zshrc"] = "/dots/zshrc"
	s := zshStep(m, out)
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || !strings.Contains(strings.Join(out.Lines, "\n"), "a link to /dots/zshrc") {
		t.Errorf("Plan = %v, %v, out %v; want nothing and a warning", p.Changes, err, out.Lines)
	}
	items, _ := s.Report(context.Background())
	if len(items) != 4 || !strings.Contains(items[3].Conflict, "a link to /dots/zshrc") {
		t.Errorf("Report = %+v; want the line's conflict", items)
	}
}

func TestZshStopsAtAConfigThatIsNoDirectory(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/.config"] = []byte("x")
	if _, err := zshStep(m, &effecttest.Out{}).Plan(context.Background()); err == nil ||
		!strings.Contains(err.Error(), "/h/.config is not a directory") {
		t.Errorf("Plan's error = %v, want that /h/.config is no directory", err)
	}
}

// A report warns of nothing: a plan says the warnings, and a report only reads.
func TestZshReportWarnsNothing(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Dirs["/h/.config"], m.Dirs["/h/.config/ws"] = 0o755, 0o700
	m.Files["/h/.config/ws/zshrc"] = zsh.Hook
	m.Files["/h/.zshrc"] = []byte("export EDITOR=vi\n")
	m.Links["/h/.zshrc"] = "/dots/zshrc"
	if _, err := zshStep(m, out).Report(context.Background()); err != nil || len(out.Lines) != 0 {
		t.Errorf("Report = %v, said %q; want no warning", err, out.Lines)
	}
}
