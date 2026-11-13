package cli

import (
	"context"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
)

// named is a step that does nothing, under a name.
type named string

func (n named) Name() string                          { return string(n) }
func (named) WaitsOn() string                         { return "" }
func (named) Plan(context.Context) (step.Plan, error) { return step.Plan{}, nil }
func (named) Apply(context.Context, step.Plan) error  { return nil }

func TestWsRunSideFindsTheRunAndNothingElse(t *testing.T) {
	find := WsRunSide(fakeProcess())
	if _, ok := find([]step.Step{named("one")}); ok {
		t.Error("found a run where none is")
	}
	if run, ok := find([]step.Step{named("one"), &stages.WsRun{Await: "x"}}); !ok || run.Await != "x" {
		t.Errorf("run = %+v, ok = %v", run, ok)
	}
}

func TestWsRunOfFindsNoneWhereNoneIs(t *testing.T) {
	if run := wsRunOf([]step.Step{named("one")}); run != nil {
		t.Errorf("wsRunOf = %v, want nil", run)
	}
}
