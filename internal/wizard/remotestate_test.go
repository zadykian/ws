package wizard

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/step"
)

// wsReport is a report of ws v1, as ws server state --json prints it.
const wsReport = `{
  "version": "v1",
  "roles": [{"role": "base", "steps": [
    {"step": "Packages", "items": [{"kind": "package", "name": "git", "want": "installed", "found": "missing"}],
     "changes": [{"kind": "install", "text": "install git"}]},
    {"step": "Later", "after": "Packages"}
  ]}]
}
`

// stateRemote returns a Remote whose state op prints report.
func stateRemote(report string) *Remote {
	m := effecttest.NewMachine()
	r := &Remote{SSH: SSH{M: m, Lock: "/t/l"}, Script: []byte("echo\n")}
	m.Results[remoteLine(r, "state")] = effecttest.Result{Out: []byte(report)}
	return r
}

func TestRemoteStateReturnsTheRoles(t *testing.T) {
	got, err := stateRemote(wsReport).State(context.Background(), "v1")
	if err != nil {
		t.Fatal(err)
	}
	want := []apply.RoleState{{Role: "base", Steps: []step.Report{
		{Step: "Packages", Items: []step.Item{{Kind: step.PackageItem, Name: "git", Want: "installed", Found: "missing"}},
			Changes: []step.Pending{{Kind: step.Install, Text: "install git"}}},
		{Step: "Later", After: "Packages"},
	}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestRemoteStateRefusesAnotherVersionsReport(t *testing.T) {
	for _, report := range []string{strings.Replace(wsReport, `"v1"`, `"v2"`, 1), `{"roles": []}`} {
		if got, err := stateRemote(report).State(context.Background(), "v1"); !errors.Is(err, ErrOtherWs) || got != nil {
			t.Errorf("%s: got %v, %v; want ErrOtherWs", report, got, err)
		}
	}
}
