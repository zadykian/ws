package ssh_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// A copy's directory that is a file stops the plan with the copy's error.
func TestSSHPlanReturnsTheErrorOfTheKeysCopy(t *testing.T) {
	m, out := mactest.AgentMachine(sshMachine(), mactest.AgentKey), &effecttest.Out{}
	m.Files["/h/"+sshcfg.KeyCopies] = []byte("not a directory\n")
	c := sshConf(m, out)
	c.Conf.IdentityPubKey = sshcfg.AgentKeys(mactest.AgentKey)[0].Pub()
	_, err := c.Step().Plan(context.Background())
	if err == nil || !strings.Contains(err.Error(), "/h/.config/ws/secretive is not a directory") {
		t.Errorf("Plan = %v; want the copy's error", err)
	}
}
