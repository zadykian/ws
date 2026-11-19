package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/roles/server/values"
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
	data, m, err := side.Values(t.Context(), local.OSMachine{})
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
	if _, m, err := side.Values(t.Context(), fake); err != nil || m != fake {
		t.Errorf("a fake machine: %v, %v; want it unchanged", m, err)
	}
}

func TestUnitPlatformIsUbuntu2604(t *testing.T) {
	ok := "NAME=\"Ubuntu\"\nID=ubuntu\nVERSION_ID=\"26.04\"\n"
	for _, tc := range []struct {
		name string
		text string
		want string
	}{
		{"ubuntu 26.04", ok, ""},
		{"quoted id", "ID=\"ubuntu\"\nVERSION_ID=26.04\n", ""},
		{"older ubuntu", "ID=ubuntu\nVERSION_ID=\"24.04\"\n", `not "ubuntu" "24.04"`},
		{"debian", "ID=debian\nVERSION_ID=\"13\"\n", `not "debian" "13"`},
		{"no file", "", `not "" ""`},
	} {
		m := effecttest.NewMachine()
		if tc.name != "no file" {
			m.Files[osRelease] = []byte(tc.text)
		}
		err := unitSide("/v.yml").Platform(m)
		switch {
		case tc.want == "" && err != nil:
			t.Errorf("%s: %v, want none", tc.name, err)
		case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
			t.Errorf("%s: %v, want %s", tc.name, err, tc.want)
		}
	}
}

func TestUnitValuesRefuseAnUnknownKey(t *testing.T) {
	fake := effecttest.NewMachine()
	fake.Files["/v.yml"] = []byte("WS_USR: u\n")
	if _, _, err := unitSide("/v.yml").Values(t.Context(), fake); err == nil {
		t.Error("an unknown key loaded")
	}
}
