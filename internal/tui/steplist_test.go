package tui

import (
	"context"
	"reflect"
	"testing"

	"github.com/zadykian/ws/step"
)

// namedStep is a step that a test lists and never runs.
type namedStep string

func (n namedStep) Name() string                            { return string(n) }
func (n namedStep) WaitsOn() string                         { return "" }
func (n namedStep) Plan(context.Context) (step.Plan, error) { return step.Plan{}, nil }
func (n namedStep) Apply(context.Context, step.Plan) error  { return nil }

func TestNewGroupNamesTheSteps(t *testing.T) {
	got := NewGroup("base", []step.Step{namedStep("Packages"), namedStep("Locale")})
	want := Group{"base", []string{"Packages", "Locale"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewGroup = %+v, want %+v", got, want)
	}
}
