package docker

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestSubIDsNeedARangeInBothFiles(t *testing.T) {
	for name, files := range map[string][2]string{
		"both":        {"ws:100000:65536\n", "ws:100000:65536\n"},
		"no subgid":   {"ws:100000:65536\n", "bob:100000:65536\n"},
		"too small":   {"ws:100000:1000\n", "ws:100000:65536\n"},
		"no subuid":   {"", "ws:100000:65536\n"},
		"a bad count": {"ws:100000:many\n", "ws:100000:65536\n"},
	} {
		m := effecttest.NewMachine()
		if files[0] != "" {
			m.Files["/etc/subuid"] = []byte(files[0])
		}
		m.Files["/etc/subgid"] = []byte(files[1])
		changes, err := subIDs{m, "ws"}.Plan(t.Context())
		if (err == nil) != (name == "both") || changes != nil {
			t.Errorf("%s: Plan = %v, %v", name, changes, err)
		}
	}
}
