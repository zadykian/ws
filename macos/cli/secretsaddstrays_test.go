package cli

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/secrets"
)

// lab comes back as ws-b with labKey, and ws-a's olderKey stays in the secrets, which known_hosts no
// longer holds. The cli prints the one line last, after the run's output.
func TestSecretsAddListsTheKeysThatKnownHostsLacks(t *testing.T) {
	p, mac := addFixture(heldBy(olderKey))
	mac.Results[hostnameLine] = effecttest.Result{Out: []byte("ws-b\n")}
	mac.Keys = []secrets.DeployKey{{ID: 7, Title: "ws-a", Key: olderKey, ReadOnly: true}}
	status, stdout, stderr := addWs(p, "", "lab")
	want := "By hand: the secrets hold " + olderKey + " under ws-a, a key of no host in known_hosts: where no " +
		"server holds it now, as after a rebuild, drop it from the secrets' recipients and deploy keys\n"
	if status != 0 || !strings.HasSuffix(stdout, want) || strings.Count(stdout, "By hand") != 1 {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want 0 and the one line %q last", status, stdout, stderr, want)
	}
	if callAt(mac.Calls, runLine) < 0 || callAt(mac.Calls, " -X DELETE ") >= 0 {
		t.Errorf("add ran nothing on lab, or deleted a deploy key that ws-b never held: %q", mac.Calls)
	}
}
