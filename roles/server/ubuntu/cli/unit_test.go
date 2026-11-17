package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func TestUnitValuesConfineToHome(t *testing.T) {
	vars := filepath.Join(t.TempDir(), "values.yml")
	if err := os.WriteFile(vars, []byte("WS_USER: u\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	side := unitSide(vars)
	if !side.Root || side.Dir != "/opt/ws/roles/server/ubuntu" {
		t.Errorf("side = %+v, want root's, at /opt/ws", side)
	}
	data, m, err := side.Values(local.OSMachine{})
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := data.(values.Values); !ok || v.User != "u" || v.Home != "/home/u" {
		t.Errorf("values = %+v, want u's", data)
	}
	if om, ok := m.(local.OSMachine); !ok || om.Confine != "/home/u" {
		t.Errorf("machine = %+v, want one confined to /home/u", m)
	}
	fake := effecttest.NewMachine()
	if _, m, err := side.Values(fake); err != nil || m != fake {
		t.Errorf("a fake machine: %v, %v; want it unchanged", m, err)
	}
}

func TestUnitValuesRefuseAnUnknownKey(t *testing.T) {
	fake := effecttest.NewMachine()
	fake.Files["/v.yml"] = []byte("WS_USR: u\n")
	if _, _, err := unitSide("/v.yml").Values(fake); err == nil {
		t.Error("an unknown key loaded")
	}
}
