package cli

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestServerFansRefusesAConfigThatFailsItsCheck(t *testing.T) {
	p := fakeProcess()
	p.Machine.(*effecttest.Machine).Files["/ws-fans.json"] = []byte(`{"interval": 0}`)
	status, _, stderr := runWs(p, "", "server", "fans", "/ws-fans.json")
	if status != 1 || !strings.HasPrefix(stderr, "ws server fans: /ws-fans.json: ") || !strings.Contains(stderr, "interval") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if status, _, _ = runWs(p, "", "server", "fans", "a", "b"); status != 2 {
		t.Errorf("two configs: status = %d, want 2", status)
	}
	if status, _, stderr = runWs(p, "", "server", "fans"); status != 1 || !strings.Contains(stderr, "/etc/ws/fans.json") {
		t.Errorf("no config: status = %d, stderr = %q", status, stderr)
	}
}
