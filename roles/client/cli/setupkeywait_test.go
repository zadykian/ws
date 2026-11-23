package cli

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/internal/mactest"
)

// applyKey runs the Secretive step of a Mac whose values.yml names no login key, until it ends or
// the wait passes, and returns what it said and its error.
func applyKey(t *testing.T, m *effecttest.Machine, wait time.Duration) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), wait)
	defer cancel()
	var out effecttest.Out
	s := secretiveKey{M: m, Out: &out, Home: mactest.AgentHome, Values: "/etc/ws/values.yml", Poll: time.Millisecond}
	changes, err := s.Plan(ctx)
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	err = changes[0].Apply(ctx)
	return strings.Join(out.Lines, "\n"), err
}

// TestSetupTakesAKeyMadeBeforeTheAgentFirstAnswersInTheWait makes the key after setup opens
// Secretive and before the wait's first look. The agent listed no key before the open, so the key
// came in this run.
func TestSetupTakesAKeyMadeBeforeTheAgentFirstAnswersInTheWait(t *testing.T) {
	_, m := unkeyed(setupRelease(miseBody), mactest.AgentKey)
	m.After = map[string]map[string]effecttest.Result{
		openLine: {mactest.AgentList: {Out: []byte(mactest.AgentKey)}}}
	if said, err := applyKey(t, m, time.Second); err != nil {
		t.Fatalf("err = %v, said %q", err, said)
	}
	if got := valuesText(m); !strings.Contains(got, "WS_IDENTITY_PUB_KEY: "+mactest.CopyOf(mactest.AgentKey)+"\n") {
		t.Errorf("values.yml = %q, want the key made in this run", got)
	}
	if exactly(m, "run "+mactest.AgentList) > exactly(m, "run "+openLine) {
		t.Errorf("setup looked at the agent first after it opened Secretive: %q", m.Calls)
	}
}

// TestSetupSaysHowToEndTheWaitWhereTheAgentFirstAnswersAfterTheOpen has no agent answer until
// setup opens Secretive, whose first answer holds a key. Setup cannot tell whether the key came in
// this run, so it takes it only by --key, and says how to end the wait.
func TestSetupSaysHowToEndTheWaitWhereTheAgentFirstAnswersAfterTheOpen(t *testing.T) {
	_, m := unkeyed(setupRelease(miseBody), mactest.AgentKey)
	m.Results[mactest.AgentList] = effecttest.Result{Out: []byte("Error connecting to agent: Connection refused\n"),
		Err: errors.New("exit status 2")}
	m.After = map[string]map[string]effecttest.Result{
		openLine: {mactest.AgentList: {Out: []byte(mactest.AgentKey)}}}
	said, err := applyKey(t, m, 50*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "stopped before Secretive held a key") {
		t.Errorf("err = %v, want the stop of the wait", err)
	}
	for _, want := range []string{"cannot tell", "Ctrl-C", "--key NAME", strings.TrimSuffix(sshName(mactest.AgentKey), ".pub")} {
		if !strings.Contains(said, want) {
			t.Errorf("the wait did not say %q: %q", want, said)
		}
	}
	if indexOf(m, "atomic /etc/ws/values.yml") >= 0 {
		t.Errorf("values.yml was written: %q", m.Calls)
	}
}
