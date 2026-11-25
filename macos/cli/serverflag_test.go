package cli

import (
	"strconv"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/values"
)

func TestServerFlagNamesTheServerOfConf(t *testing.T) {
	m := effecttest.NewMachine()
	for slot, name := range []string{"ws", "box"} {
		m.Files[values.ServerPath("/etc/ws/values.yml", name)] = []byte(values.Text("WS_SLOT", strconv.Itoa(slot)))
	}
	m.Files["/etc/ws/values.yml"] = nil
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = withServerDirs(m)
	if status, stdout, _ := runWs(p, "macos", "values", "print", "--server", "box"); status != 0 || !strings.HasPrefix(stdout, "WS_TARGET='lan'\nWS_HOST=''\nWS_SLOT='1'\n") {
		t.Errorf("--server box: status = %d, stdout = %q", status, stdout)
	}
	if status, _, stderr := runWs(p, "macos", "values", "print"); status != 1 || !strings.Contains(stderr, "several servers") {
		t.Errorf("no flag: status = %d, stderr = %q", status, stderr)
	}
}
