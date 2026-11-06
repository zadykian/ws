package main

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/internal/wizard"
)

// reportingWs is a far ws whose script gives its status and its report, at version.
type reportingWs struct {
	farWs
	version string
}

func newReportingWs(m *effecttest.Machine, version string) reportingWs {
	return reportingWs{newFarWs(m, ""), version}
}

func (w reportingWs) RunInput(ctx context.Context, in []byte, name string, args ...string) ([]byte, error) {
	out, err := w.farWs.RunInput(ctx, in, name, args...)
	switch {
	case err != nil:
		return nil, err
	case slices.Contains(args, "status"):
		return []byte("ws " + w.version + "\n"), nil
	case slices.Contains(args, "state"):
		return json.Marshal(apply.StateDoc{Version: w.version, Roles: []apply.RoleState{{Role: "base"}}})
	}
	return out, nil
}

func TestWsReadGivesTheRolesAndReadsWsAgain(t *testing.T) {
	p, m := macTerminal()
	ws := newReportingWs(m, p.version)
	p.machine = ws
	reach := &varsReach{p: p, sudo: &wizard.Sudo{}}
	defer reach.close(context.Background())
	read := wsRead(reach, p.version)
	for range 2 {
		sections, err := read(t.Context(), nil)
		if err != nil || len(sections) != 1 || sections[0].Name != "base" {
			t.Fatalf("read = %+v, %v", sections, err)
		}
	}
	var ops []string
	for range 4 {
		ops = append(ops, strings.Join(<-ws.ops, " "))
	}
	if want := "status"; ops[0] != want || ops[2] != want {
		t.Errorf("ops = %q", ops)
	}
	if got := lockCalls(m.Calls, "mkdir"); got != 1 {
		t.Errorf("took the lock %d times", got)
	}
}

func TestWsReadRefusesAnotherVersion(t *testing.T) {
	p, m := macTerminal()
	p.machine = newReportingWs(m, "v0.0.1")
	reach := &varsReach{p: p, sudo: &wizard.Sudo{}}
	defer reach.close(context.Background())
	if _, err := wsRead(reach, p.version)(t.Context(), nil); err == nil || !strings.Contains(err.Error(), "holds v0.0.1") {
		t.Errorf("err = %v", err)
	}
}
