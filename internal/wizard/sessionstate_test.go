package wizard

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
)

// wsReport is a report of ws v1, as ws server ubuntu state --json prints it.
const wsReport = `{
  "version": "v1",
  "roles": [{"role": "base", "steps": [
    {"step": "Packages", "items": [{"kind": "package", "name": "git", "want": "installed", "found": "missing"}],
     "changes": [{"kind": "install", "text": "install git"}]},
    {"step": "Later", "after": "Packages"}
  ]}]
}
`

// stateSession returns a session on a machine whose status names held as ws's version, and whose
// state op prints report.
func stateSession(held, report string) (*Session, *effecttest.Machine) {
	m := effecttest.NewMachine()
	r := NewSession(fakeReach{Reach: MacReach{}, M: m}, Scripts{Remote: []byte("echo\n")})
	m.Results[opLine("status")] = effecttest.Result{Out: wsStatus{Version: held}.text()}
	m.Results[opLine("state")] = effecttest.Result{Out: []byte(report)}
	return r, m
}

// ranOp counts the calls of the remote script's operation.
func ranOp(m *effecttest.Machine, op string) int {
	n := 0
	for _, c := range m.Calls {
		if strings.HasSuffix(c, "op "+op) {
			n++
		}
	}
	return n
}

func TestSessionStateReturnsTheRoles(t *testing.T) {
	r, _ := stateSession("v1", wsReport)
	got, err := r.State(context.Background(), "v1")
	if err != nil {
		t.Fatal(err)
	}
	want := []role.RoleState{{Role: "base", Steps: []step.Report{
		{Step: "Packages", Items: []step.Item{{Kind: step.PackageItem, Name: "git", Want: "installed", Found: "missing"}},
			Changes: []step.Pending{{Kind: step.Install, Text: "install git"}}},
		{Step: "Later", After: "Packages"},
	}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestSessionStateLeavesWhatStatusRead(t *testing.T) {
	r, m := stateSession("v1", wsReport)
	for range 2 {
		if _, err := r.State(context.Background(), "v1"); err != nil {
			t.Fatal(err)
		}
	}
	if status, state := ranOp(m, "status"), ranOp(m, "state"); status != 1 || state != 2 {
		t.Errorf("status ran %d times and state %d; want 1 and 2", status, state)
	}
}

func TestSessionStateRefusesAnotherWs(t *testing.T) {
	cases := []struct {
		name, held, report string
		state              int
	}{
		{"none yet", "-", "", 0},
		{"another by status", "v0", "", 0},
		{"another by its report", "v1", strings.Replace(wsReport, `"v1"`, `"v2"`, 1), 1},
		{"a report with no version", "v1", `{"roles": []}`, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, m := stateSession(c.held, c.report)
			if got, err := r.State(context.Background(), "v1"); !errors.Is(err, ErrOtherWs) || got != nil {
				t.Errorf("got %v, %v; want ErrOtherWs", got, err)
			}
			if n := ranOp(m, "state"); n != c.state {
				t.Errorf("state ran %d times; want %d", n, c.state)
			}
		})
	}
}

func TestSessionStateFails(t *testing.T) {
	cases := []struct {
		name   string
		result effecttest.Result
	}{
		{"the op fails", effecttest.Result{Err: errors.New("denied")}},
		{"cut short", effecttest.Result{Out: []byte(wsReport[:40])}},
		{"a line beside the report", effecttest.Result{Out: []byte("a line\n" + wsReport)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, m := stateSession("v1", "")
			m.Results[opLine("state")] = c.result
			if got, err := r.State(context.Background(), "v1"); err == nil || errors.Is(err, ErrOtherWs) || got != nil {
				t.Errorf("got %v, %v; want a failure that is no refusal", got, err)
			}
		})
	}
}

func TestSessionStateWithNoStatus(t *testing.T) {
	r, m := stateSession("v1", wsReport)
	m.Results[opLine("status")] = effecttest.Result{Err: errors.New("denied")}
	if got, err := r.State(context.Background(), "v1"); err == nil || errors.Is(err, ErrOtherWs) || got != nil {
		t.Errorf("got %v, %v; want a failure that is no refusal", got, err)
	}
	if n := ranOp(m, "state"); n != 0 {
		t.Errorf("state ran %d times", n)
	}
}
