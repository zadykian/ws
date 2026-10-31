package step_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/step"
)

func TestRun(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name        string
		check       bool
		steps       func(log *[]string) []step.Step
		wantChanged bool
		wantErr     string
		wantLog     []string
		wantOut     []string
	}{
		{
			name: "apply plans, applies and plans again",
			steps: func(l *[]string) []step.Step {
				return []step.Step{
					&fake{name: "a", todo: []step.Change{change("x")}, log: l},
					&fake{name: "b", log: l},
				}
			},
			wantChanged: true,
			wantLog:     []string{"plan a", "apply a", "plan a", "plan b"},
			wantOut:     []string{"say: a: applying 1 change(s)"},
		},
		{
			name:  "check prints and applies nothing",
			check: true,
			steps: func(l *[]string) []step.Step {
				return []step.Step{&fake{name: "a", todo: []step.Change{change("x"), change("y")}, log: l}}
			},
			wantChanged: true,
			wantLog:     []string{"plan a"},
			wantOut:     []string{"say: a: x", "say: a: y"},
		},
		{
			name:    "check of a done step is silent",
			check:   true,
			steps:   func(l *[]string) []step.Step { return []step.Step{&fake{name: "a", log: l}} },
			wantLog: []string{"plan a"},
		},
		{
			name: "a plan not empty after apply",
			steps: func(l *[]string) []step.Step {
				return []step.Step{&fake{name: "a", todo: []step.Change{change("x")}, sticky: true, log: l}}
			},
			wantChanged: true,
			wantErr:     "a: still has 1 change(s) after applying, the first: x",
			wantLog:     []string{"plan a", "apply a", "plan a"},
			wantOut:     []string{"say: a: applying 1 change(s)"},
		},
		{
			name:  "check says a step waits",
			check: true,
			steps: func(l *[]string) []step.Step {
				return []step.Step{
					&fake{name: "a", todo: []step.Change{change("x")}, log: l},
					&fake{name: "b", todo: []step.Change{change("y")}, waitsOn: "a", log: l},
				}
			},
			wantChanged: true,
			wantLog:     []string{"plan a"},
			wantOut:     []string{"say: a: x", "say: b: waits on a"},
		},
		{
			name:  "check skips a step whose plan would fail",
			check: true,
			steps: func(l *[]string) []step.Step {
				return []step.Step{
					&fake{name: "a", todo: []step.Change{change("x")}, log: l},
					&fake{name: "b", waitsOn: "a", planErr: errors.New("no gh"), log: l},
					&fake{name: "c", waitsOn: "b", log: l},
				}
			},
			wantChanged: true,
			wantLog:     []string{"plan a"},
			wantOut:     []string{"say: a: x", "say: b: waits on a", "say: c: waits on b"},
		},
		{
			name:  "check of a step whose awaited step is done",
			check: true,
			steps: func(l *[]string) []step.Step {
				return []step.Step{
					&fake{name: "a", log: l},
					&fake{name: "b", todo: []step.Change{change("y")}, waitsOn: "a", log: l},
				}
			},
			wantChanged: true,
			wantLog:     []string{"plan a", "plan b"},
			wantOut:     []string{"say: b: y"},
		},
		{
			name: "a wait on a later step",
			steps: func(l *[]string) []step.Step {
				return []step.Step{
					&fake{name: "a", waitsOn: "b", log: l},
					&fake{name: "b", log: l},
				}
			},
			wantErr: `a waits on "b", which is not an earlier step`,
		},
		{
			name: "a wait on itself",
			steps: func(l *[]string) []step.Step {
				return []step.Step{&fake{name: "a", waitsOn: "a", log: l}}
			},
			wantErr: `a waits on "a", which is not an earlier step`,
		},
		{
			name: "a plan fails",
			steps: func(l *[]string) []step.Step {
				return []step.Step{&fake{name: "a", planErr: boom, log: l}}
			},
			wantErr: "a: planning: boom",
			wantLog: []string{"plan a"},
		},
		{
			name: "an apply fails",
			steps: func(l *[]string) []step.Step {
				return []step.Step{
					&fake{name: "a", todo: []step.Change{change("x")}, applyErr: boom, log: l},
					&fake{name: "b", log: l},
				}
			},
			wantChanged: true,
			wantErr:     "a: applying: boom",
			wantLog:     []string{"plan a", "apply a"},
			wantOut:     []string{"say: a: applying 1 change(s)"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var log []string
			out := &effecttest.Out{}
			changed, err := step.Run(context.Background(), tt.steps(&log), out, tt.check)
			if changed != tt.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tt.wantChanged)
			}
			if tt.wantErr == "" && err != nil || tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Errorf("err = %v, want %q", err, tt.wantErr)
			}
			if !slices.Equal(log, tt.wantLog) {
				t.Errorf("calls = %q, want %q", log, tt.wantLog)
			}
			if !slices.Equal(out.Lines, tt.wantOut) {
				t.Errorf("out = %q, want %q", out.Lines, tt.wantOut)
			}
		})
	}
}
