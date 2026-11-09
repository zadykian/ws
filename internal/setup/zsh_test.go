package setup_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/mac"
	"github.com/zadykian/ws/step"
)

const zshSource = "source ~/.config/ws/zshrc\n"

func zshStep(m *effecttest.Machine, out *effecttest.Out) *setup.Zsh {
	return &setup.Zsh{Home: "/h", Hook: mac.Zshrc, M: m, Out: out,
		Now: func() time.Time { return time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC) }}
}

// The second plan finds mac/zshrc's header, so the hook that ws embeds is its own.
func TestZshMakesTheHookAndZshrcThenNothing(t *testing.T) {
	m := effecttest.NewMachine()
	s := zshStep(m, &effecttest.Out{})
	p, err := s.Plan(context.Background())
	want := []step.Change{setup.MakeDir{Path: "/h/.config", Mode: 0o755}, setup.MakeDir{Path: "/h/.config/ws", Mode: 0o700},
		setup.Write{Path: "/h/.config/ws/zshrc", After: mac.Zshrc, Mode: 0o644},
		setup.Write{Path: "/h/.zshrc", After: []byte(zshSource), Mode: 0o644}}
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
	if err != nil || len(items) != 2 || items[0].Conflict != "" || items[1].Found != items[1].Want {
		t.Errorf("Report = %+v, %v; want the hook and the line, no conflict", items, err)
	}
}

func TestZshAddsTheLineAtTheEndAndKeepsTheOldZshrc(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/h/.config"], m.Dirs["/h/.config/ws"] = 0o755, 0o700
	m.Files["/h/.config/ws/zshrc"] = mac.Zshrc
	m.Files["/h/.zshrc"] = []byte("export EDITOR=vi")
	p, err := zshStep(m, &effecttest.Out{}).Plan(context.Background())
	want := []step.Change{setup.Backup{Path: "/h/.zshrc", To: "/h/.zshrc.20261109100000~"},
		setup.Write{Path: "/h/.zshrc", Before: []byte("export EDITOR=vi"),
			After: []byte("export EDITOR=vi\n" + zshSource), Exists: true, Mode: 0o644}}
	if err != nil || !reflect.DeepEqual(p.Changes, want) {
		t.Errorf("Plan = %v, %v; want %v", p.Changes, err, want)
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
	m.Files["/h/.config/ws/zshrc"] = mac.Zshrc
	m.Files["/h/.zshrc"] = []byte("export EDITOR=vi\n")
	m.Links["/h/.zshrc"] = "/dots/zshrc"
	s := zshStep(m, out)
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || !strings.Contains(strings.Join(out.Lines, "\n"), "a link to /dots/zshrc") {
		t.Errorf("Plan = %v, %v, out %v; want nothing and a warning", p.Changes, err, out.Lines)
	}
	items, _ := s.Report(context.Background())
	if len(items) != 2 || !strings.Contains(items[1].Conflict, "a link to /dots/zshrc") {
		t.Errorf("Report = %+v; want the line's conflict", items)
	}
}
