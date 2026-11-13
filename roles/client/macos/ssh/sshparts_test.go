package ssh

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

func partsOf(m *effecttest.Machine) SSH {
	m.Files["/h/.ssh/id_ed25519.pub"] = []byte("ssh-ed25519 AAAA me\n")
	m.Results["ssh-keygen -l -f /h/.ssh/id_ed25519.pub"] = effecttest.Result{}
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/tmp/x\n")}
	m.Results["rm -rf /tmp/x"] = effecttest.Result{}
	m.Results["ssh -G -F /tmp/x/new ws"] = effecttest.Result{}
	m.Results["ssh -G -F /tmp/x/whole ws"] = effecttest.Result{}
	return SSH{
		Conf: conf.Conf{User: "me", IdentityPubKey: "~/.ssh/id_ed25519.pub"}, Home: "/h", M: m, Out: &effecttest.Out{}}
}

func TestSSHPartsPlanTheChangesOfAFreshHomeInOrder(t *testing.T) {
	s := partsOf(effecttest.NewMachine())
	p, err := resource.Lazy{Title: "ssh", Parts: s.parts}.Plan(t.Context())
	if err != nil || len(p.Changes) != 4 || !strings.HasPrefix(p.Changes[2].String(), "--- /dev/null\n+++ /h/.ssh/config.d/ws") {
		t.Errorf("Plan = %v, %v; want 4 changes, the third config.d/ws's diff", p.Changes, err)
	}
}
