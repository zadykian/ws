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

const joinScript = "#!/bin/sh\n# Written by ws mac apply: its next run replaces changes made here.\necho\n"

// The second plan finds mac/ws-join's header, so the script that ws embeds is its own.
func TestJoinMakesTheDirsAndTheFileThenNothing(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := &setup.Join{Home: "/h", Script: mac.Join, M: m, Out: out}
	p, err := s.Plan(context.Background())
	want := []step.Change{setup.MakeDir{Path: "/h/.local", Mode: 0o755}, setup.MakeDir{Path: "/h/.local/bin", Mode: 0o755},
		setup.Write{Path: "/h/.local/bin/ws-join", After: mac.Join, Mode: 0o755}}
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
		t.Errorf("Report = %+v, %v; want ws-join with no conflict", items, err)
	}
}

func TestJoinSetsTheModeAndReplacesItsOwnFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/h/.local"], m.Dirs["/h/.local/bin"] = 0o755, 0o755
	m.Files["/h/.local/bin/ws-join"] = []byte(joinScript)
	s := &setup.Join{Home: "/h", Script: []byte(joinScript), M: m, Out: &effecttest.Out{}}
	p, _ := s.Plan(context.Background())
	if want := []step.Change{setup.SetMode{Path: "/h/.local/bin/ws-join", Mode: 0o755}}; !reflect.DeepEqual(p.Changes, want) {
		t.Errorf("Plan = %v; want %v", p.Changes, want)
	}
	m.Files["/h/.local/bin/ws-join"] = []byte(joinScript + "echo again\n")
	p, _ = s.Plan(context.Background())
	if len(p.Changes) != 1 || !p.Changes[0].(setup.Write).Exists {
		t.Errorf("Plan = %v; want ws's file written again", p.Changes)
	}
}

func TestJoinLeavesAFileNotItsOwn(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Dirs["/h/.local"], m.Dirs["/h/.local/bin"] = 0o755, 0o755
	m.Files["/h/.local/bin/ws-join"] = []byte("#!/bin/sh\nexec ssh ws\n")
	s := &setup.Join{Home: "/h", Script: []byte(joinScript), M: m, Out: out}
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || !strings.Contains(strings.Join(out.Lines, "\n"), "is not ws's") {
		t.Errorf("Plan = %v, %v, out %v; want nothing and a warning", p.Changes, err, out.Lines)
	}
	items, _ := s.Report(context.Background())
	if len(items) != 1 || !strings.Contains(items[0].Conflict, "not ws's") {
		t.Errorf("Report = %+v; want a conflict", items)
	}
}
