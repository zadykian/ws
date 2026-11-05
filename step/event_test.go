package step_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

func TestPrinter(t *testing.T) {
	two := []step.Change{change("x"), change("y")}
	tests := []struct {
		name string
		e    step.Event
		want []string
	}{
		{"a plan under check", step.Event{Kind: step.Planned, Step: "a", Check: true, Changes: two}, []string{"say: a: x", "say: a: y"}},
		{"a plan to apply", step.Event{Kind: step.Planned, Step: "a", Changes: two}, nil},
		{"a start", step.Event{Kind: step.Started, Step: "a", Changes: two}, []string{"say: a: applying 2 change(s)"}},
		{"a step done", step.Event{Kind: step.Done, Step: "a", Changes: two}, nil},
		{"a step failed", step.Event{Kind: step.Failed, Step: "a", Changes: two}, nil},
		{"a step behind another", step.Event{Kind: step.Skipped, Step: "b", After: "a"}, []string{"say: b: after a"}},
		{"a step past a failure", step.Event{Kind: step.Skipped, Step: "b"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &effecttest.Out{}
			step.Printer{Out: out}.Observe(tt.e)
			if !slices.Equal(out.Lines, tt.want) {
				t.Errorf("lines = %q, want %q", out.Lines, tt.want)
			}
		})
	}
}
