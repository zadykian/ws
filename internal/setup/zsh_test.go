package setup_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/mac"
	"github.com/zadykian/ws/step"
)

func zshStep(m *effecttest.Machine, out *effecttest.Out) *setup.Zsh {
	return &setup.Zsh{Home: "/h", Hook: mac.Zshrc, M: m, Out: out}
}

// The second plan finds mac/zshrc's header, so the hook that ws embeds is its own.
func TestZshMakesTheHookThenNothing(t *testing.T) {
	m := effecttest.NewMachine()
	s := zshStep(m, &effecttest.Out{})
	p, err := s.Plan(context.Background())
	want := []step.Change{setup.MakeDir{Path: "/h/.config", Mode: 0o755}, setup.MakeDir{Path: "/h/.config/ws", Mode: 0o700},
		setup.Write{Path: "/h/.config/ws/zshrc", After: mac.Zshrc, Mode: 0o644}}
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
	if err != nil || len(items) != 1 || items[0].Conflict != "" {
		t.Errorf("Report = %+v, %v; want the hook with no conflict", items, err)
	}
}

func TestZshLeavesAHookNotItsOwn(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Dirs["/h/.config"], m.Dirs["/h/.config/ws"] = 0o755, 0o700
	m.Files["/h/.config/ws/zshrc"] = []byte("echo mine\n")
	p, err := zshStep(m, out).Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || !strings.Contains(strings.Join(out.Lines, "\n"), "is not ws's") {
		t.Errorf("Plan = %v, %v, out %v; want nothing and a warning", p.Changes, err, out.Lines)
	}
}
