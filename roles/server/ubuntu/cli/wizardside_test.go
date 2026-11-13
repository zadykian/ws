package cli

import (
	"testing"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
)

func TestWsRunSideFindsTheRunAndNothingElse(t *testing.T) {
	find := WsRunSide(fakeProcess())
	if _, ok := find([]step.Step{stepOf{name: "one"}}); ok {
		t.Error("found a run where none is")
	}
	if run, ok := find([]step.Step{stepOf{name: "one"}, &stages.WsRun{Await: "x"}}); !ok || run.Await != "x" {
		t.Errorf("run = %+v, ok = %v", run, ok)
	}
}

func TestWsRunOfFindsNoneWhereNoneIs(t *testing.T) {
	if run := wsRunOf([]step.Step{stepOf{name: "one"}}); run != nil {
		t.Errorf("wsRunOf = %v, want nil", run)
	}
}
