package step_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

// journal keeps each event as a line: kind, parent/step#index, the size of the plan, and the rest.
type journal struct{ lines []string }

func (j *journal) Observe(e step.Event) {
	kind := []string{"planned", "started", "done", "failed", "skipped"}[e.Kind]
	line := fmt.Sprintf("%s %s/%s#%d %d", kind, e.Parent, e.Step, e.Index, len(e.Changes))
	if e.After != "" {
		line += " after " + e.After
	}
	if e.Err != nil {
		line += " " + e.Err.Error()
	}
	j.lines = append(j.lines, line)
}

func TestRunWithReportsEachStep(t *testing.T) {
	x := []step.Change{change("x")}
	tests := []struct {
		name  string
		check bool
		steps func(log *[]string) []step.Step
		want  []string
	}{
		{name: "a step with nothing to make", steps: func(l *[]string) []step.Step {
			return []step.Step{&fake{name: "a", log: l}}
		}, want: []string{"planned p/a#0 0", "done p/a#0 0"}},
		{name: "a step that applies", steps: func(l *[]string) []step.Step {
			return []step.Step{&fake{name: "a", todo: x, log: l}}
		}, want: []string{"planned p/a#0 1", "started p/a#0 1", "done p/a#0 1"}},
		{name: "a check", check: true, steps: func(l *[]string) []step.Step {
			return []step.Step{&fake{name: "a", todo: x, log: l}, &fake{name: "b", waitsOn: "a", log: l}}
		}, want: []string{"planned p/a#0 1", "done p/a#0 1", "skipped p/b#1 0 after a"}},
		{name: "a name that comes twice", steps: func(l *[]string) []step.Step {
			return []step.Step{&fake{name: "a", log: l}, &fake{name: "a", log: l}}
		}, want: []string{"planned p/a#0 0", "done p/a#0 0", "planned p/a#1 0", "done p/a#1 0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var log []string
			j := &journal{}
			if _, err := step.RunWith(context.Background(), tt.steps(&log), step.Options{Observer: j, Check: tt.check, Parent: "p"}); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(j.lines, tt.want) {
				t.Errorf("events = %q, want %q", j.lines, tt.want)
			}
		})
	}
}

func TestRunWithEndsEveryStepOfAFailedRun(t *testing.T) {
	boom := errors.New("boom")
	x := []step.Change{change("x")}
	tests := []struct {
		name  string
		first fake
		want  []string
	}{
		{"a plan fails", fake{planErr: boom}, []string{"failed /a#0 0 a: planning: boom"}},
		{"an apply fails", fake{todo: x, applyErr: boom},
			[]string{"planned /a#0 1", "started /a#0 1", "failed /a#0 1 a: applying: boom"}},
		{"a plan not empty after apply", fake{todo: x, sticky: true}, []string{"planned /a#0 1", "started /a#0 1",
			"failed /a#0 1 a: still has 1 change(s) after applying, the first: x"}},
		{"a wait on a later step", fake{waitsOn: "c"},
			[]string{`failed /a#0 0 a waits on "c", which is not an earlier step`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var log []string
			first := tt.first
			first.name, first.log = "a", &log
			steps := []step.Step{&first, &fake{name: "b", log: &log}, &fake{name: "c", log: &log}}
			j := &journal{}
			if _, err := step.RunWith(context.Background(), steps, step.Options{Observer: j}); err == nil {
				t.Fatal("the run succeeded")
			}
			want := slices.Concat(tt.want, []string{"skipped /b#1 0", "skipped /c#2 0"})
			if !slices.Equal(j.lines, want) {
				t.Errorf("events = %q, want %q", j.lines, want)
			}
		})
	}
}

func TestUnreachedEndsEachStep(t *testing.T) {
	j := &journal{}
	step.Unreached([]step.Step{&fake{name: "a"}, &fake{name: "b"}}, step.Options{Observer: j, Parent: "role"})
	if want := []string{"skipped role/a#0 0", "skipped role/b#1 0"}; !slices.Equal(j.lines, want) {
		t.Errorf("events = %q, want %q", j.lines, want)
	}
}

func TestRunWithTakesNoObserver(t *testing.T) {
	var log []string
	if _, err := step.RunWith(context.Background(), []step.Step{&fake{name: "a", log: &log}}, step.Options{}); err != nil {
		t.Error(err)
	}
}

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
