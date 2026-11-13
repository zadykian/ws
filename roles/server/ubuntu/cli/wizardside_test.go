package cli

import (
	"testing"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu"
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

func TestNestedRowsAreTheRolesUnread(t *testing.T) {
	names := ubuntu.Names()
	rows := NestedRows((&stages.WsRun{}).Name())
	if len(rows) != len(names) {
		t.Fatalf("NestedRows holds %d rows, want one a role (%d)", len(rows), len(names))
	}
	for i, row := range rows {
		if row.Name != names[i] || len(row.Steps) != 0 {
			t.Errorf("row %d = %+v, want %q with no steps", i, row, names[i])
		}
	}
	if other := NestedRows("another"); other != nil {
		t.Errorf("NestedRows of another stage = %+v, want nil", other)
	}
}
