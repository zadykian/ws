package setup_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/mac"
	"github.com/zadykian/ws/step"
)

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
