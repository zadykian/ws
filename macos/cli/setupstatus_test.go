package cli

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// statusTableLine is the table that names what differs, shown after a difference.
const statusTableLine = miseBase + " status --missing"

func TestSetupPlanAsksMisesStatusAfterAPlanWithNothingToDo(t *testing.T) {
	report := `{"defaults": [{"state": "differs"}]}`
	for name, tc := range map[string]struct {
		result effecttest.Result
		status int
		table  bool
	}{
		"in sync":        {effecttest.Result{Out: []byte("{}")}, 0, false},
		"a difference":   {effecttest.Result{Out: []byte(report), Err: miseExit(1)}, 2, true},
		"an error":       {effecttest.Result{Err: miseExit(1)}, 1, false},
		"text, no JSON":  {effecttest.Result{Out: []byte("Error: no config"), Err: miseExit(1)}, 1, false},
		"another status": {effecttest.Result{Out: []byte(report), Err: miseExit(3)}, 1, false},
		"a lost mise":    {effecttest.Result{Err: errors.New("no mise")}, 1, false},
	} {
		t.Run(name, func(t *testing.T) {
			rel := setupRelease(miseBody)
			p, m := setupProcess(rel)
			standing(m, rel)
			m.Results[planLine] = effecttest.Result{}
			m.Results[statusLine] = tc.result
			m.Results[statusTableLine] = effecttest.Result{Err: miseExit(1)}
			status, stderr := runSetup(p, rel, "--plan")
			if status != tc.status {
				t.Errorf("status = %d, want %d, stderr = %q", status, tc.status, stderr)
			}
			if shown := slices.Contains(m.Calls, "run "+statusTableLine); shown != tc.table {
				t.Errorf("the table of what differs was shown = %v, want %v", shown, tc.table)
			}
			if tc.table && !strings.Contains(stderr, "differs from the config") {
				t.Errorf("stderr = %q", stderr)
			}
			noWrites(t, m)
		})
	}
}

func TestSetupPlanAsksNoStatusWhereThePlanHasAChange(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	standing(m, rel)
	m.Results[planLine] = effecttest.Result{Err: miseExit(2)}
	if status, _ := runSetup(p, rel, "--plan"); status != 2 {
		t.Errorf("status = %d, want 2", status)
	}
	if indexOf(m, "run "+miseBase+" status") >= 0 {
		t.Errorf("the status ran: %q", m.Calls)
	}
}
