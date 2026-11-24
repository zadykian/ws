package cli

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// The host's sshd and its fail2ban are the guest's, and it holds the units of both.
func TestHostHoldsSecuritysUnits(t *testing.T) {
	host := hostSpecs(t, effecttest.NewMachine())
	for _, name := range []string{"guard/security-keys", "service/security-sshd"} {
		if _, ok := host[name]; !ok {
			t.Errorf("the host lacks %s", name)
		}
	}
}
