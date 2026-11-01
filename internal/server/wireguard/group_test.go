package wireguard

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestGroupIDReadsEtcGroup(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/etc/group"] = []byte("root:x:0:\nsystemd-timesync:x:996:\nsystemd-network:x:998:\n")
	if id, err := groupID(m); err != nil || id != 998 {
		t.Errorf("groupID = %d, %v; want 998", id, err)
	}
}

func TestGroupIDRefusesAMissingGroup(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/etc/group"] = []byte("root:x:0:\n")
	if _, err := groupID(m); err == nil || !strings.Contains(err.Error(), "systemd-network is missing") {
		t.Errorf("groupID = %v, want a refusal", err)
	}
}
