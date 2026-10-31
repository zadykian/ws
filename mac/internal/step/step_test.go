package step_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/step"
)

func TestRun(t *testing.T) {
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
