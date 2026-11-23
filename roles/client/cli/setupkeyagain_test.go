package cli

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/roles/client/internal/mactest"
	"github.com/zadykian/ws/roles/client/values"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// TestSetupTakesKeyAgainForTheKeyThatValuesName runs setup --key again once an earlier run named
// the key, as a retry after a failed bootstrap does. The same key passes, by its name or its
// fingerprint, and another is refused.
func TestSetupTakesKeyAgainForTheKeyThatValuesName(t *testing.T) {
	rel := setupRelease(miseBody)
	gitName := strings.TrimSuffix(sshName(mactest.AgentGit), ".pub")
	for name, pick := range map[string]string{
		"its name":        gitName,
		"its name by pub": gitName + ".pub",
		"its fingerprint": sshcfg.AgentKey{Line: strings.TrimSpace(mactest.AgentGit)}.Fingerprint(),
	} {
		t.Run(name, func(t *testing.T) {
			p, m := setupProcess(rel)
			standing(m, rel)
			mactest.AgentMachine(m, mactest.AgentKey, mactest.AgentGit)
			named := values.Text("WS_IDENTITY_PUB_KEY", mactest.CopyOf(mactest.AgentGit), "WS_IDENTITY_AGENT", secretiveAgent)
			writeConfs(m, named, values.Text("WS_SLOT", "0"))
			if status, stderr := runSetup(p, rel, "--key", pick); status != 0 {
				t.Fatalf("status = %d, stderr = %q", status, stderr)
			}
			if got := valuesText(m); got != named {
				t.Errorf("values.yml = %q, want it as it was, %q", got, named)
			}
			if indexOf(m, "run "+installLine) >= 0 || exactly(m, "run "+miseLine) < 0 {
				t.Errorf("want no Secretive step and the bootstrap: %q", m.Calls)
			}
		})
	}
	t.Run("another key", func(t *testing.T) {
		p, m := setupProcess(rel)
		standing(m, rel)
		mactest.AgentMachine(m, mactest.AgentKey, mactest.AgentGit)
		writeConfs(m, values.Text("WS_IDENTITY_PUB_KEY", mactest.CopyOf(mactest.AgentGit)), values.Text("WS_SLOT", "0"))
		other := strings.TrimSuffix(sshName(mactest.AgentKey), ".pub")
		status, stderr := runSetup(p, rel, "--key", other)
		if status != 1 || !strings.Contains(stderr, "names WS_IDENTITY_PUB_KEY already") ||
			!strings.Contains(stderr, mactest.CopyOf(mactest.AgentGit)) {
			t.Errorf("status = %d, stderr = %q", status, stderr)
		}
		noWrites(t, m)
		if exactly(m, "run "+miseLine) >= 0 {
			t.Errorf("the bootstrap ran: %q", m.Calls)
		}
	})
}
