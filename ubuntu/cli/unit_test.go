package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/ubuntu/values"
)

func TestUnitValuesLoadTheFile(t *testing.T) {
	vars := filepath.Join(t.TempDir(), "values.yml")
	if err := os.WriteFile(vars, []byte("WS_BASE_LOCALE: C\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	side := unitSide(vars)
	if !side.Root || side.Dir != "/opt/ws/ubuntu/roles" {
		t.Errorf("side = %+v, want root's, at /opt/ws", side)
	}
	data, err := side.Values(t.Context(), local.OSMachine{})
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := data.(values.Values); !ok || v.Base.Locale != "C" {
		t.Errorf("values = %+v, want the file's", data)
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
	if _, err := unitSide("/v.yml").Values(t.Context(), fake); err == nil {
		t.Error("an unknown key loaded")
	}
}

func TestUnitRunDirIsWhereSetupBootstraps(t *testing.T) {
	if runDir != guestSetup.dir() || runDir != "/opt/ws/ubuntu/guest" {
		t.Errorf("runDir = %q, the guest's setup runs in %q, want both /opt/ws/ubuntu/guest", runDir, guestSetup.dir())
	}
	if hostRunDir != hostSetup.dir() || hostRunDir != "/opt/ws/ubuntu/host" {
		t.Errorf("hostRunDir = %q, the host's setup runs in %q, want both /opt/ws/ubuntu/host", hostRunDir, hostSetup.dir())
	}
}

func TestHostSideLoadsTheHostsValues(t *testing.T) {
	side := hostSide("/v.yml")
	if !side.Root || side.RoleFile != "mise.host.toml" || side.RunDir != "/opt/ws/ubuntu/host" || side.Dir != unitsDir {
		t.Errorf("side = %+v, want root's, at /opt/ws, run in ubuntu/host", side)
	}
	fake := effecttest.NewMachine()
	fake.Files["/v.yml"] = []byte("WS_BASE_LOCALE: C\n")
	data, err := side.Values(t.Context(), fake)
	if v, ok := data.(values.Host); err != nil || !ok || v.Base.Locale != "C" {
		t.Errorf("values = %+v, %v, want the host's, with the file's locale", data, err)
	}
	fake.Files["/v.yml"] = []byte("WS_ZED_NOFILE: 1\n")
	if _, err := side.Values(t.Context(), fake); err == nil {
		t.Error("a guest's key loaded on the host")
	}
}
