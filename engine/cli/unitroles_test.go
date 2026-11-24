package cli_test

import (
	"os"
	"path/filepath"
	"testing"
)

// roleSpecs holds a role of each side's alone, and one of both, with the host's unit of that one.
var roleSpecs = map[string]string{
	"g/units.toml.tmpl": fileUnit("g", "/etc/g", ""), "g/g": "1\n",
	"h/units.toml.tmpl": fileUnit("h", "/etc/h", ""), "h/h": "1\n",
	"both/units.toml.tmpl": fileUnit("c", "/etc/c", "") + "[file.d]\npath = \"/etc/d\"\nsource = \"c\"\nwhen = { side = \"host\" }\n",
	"both/c":               "1\n",
}

func TestUnitListsTheRolesOfItsSide(t *testing.T) {
	for side, want := range map[string]string{"guest": "file/c\nfile/g\n", "host": "file/c\nfile/d\nfile/h\n"} {
		e, sides := twoSides(t, side, roleSpecs)
		dir := e.Getenv("WS_UNITS")
		for i, file := range []string{"guest", "host"} {
			sides[i].Name, sides[i].RoleFile = file, "mise."+file+".toml"
			for _, role := range map[string][]string{"guest": {"g", "both"}, "host": {"h", "both"}}[file] {
				if err := os.WriteFile(filepath.Join(dir, role, sides[i].RoleFile), nil, 0o600); err != nil {
					t.Fatal(err)
				}
			}
		}
		if status, stdout, stderr := runSides(e, sides, "list"); status != 0 || stdout != want {
			t.Errorf("in the %s's run directory: status = %d, stdout = %q, stderr = %q; want %q", side, status, stdout, stderr, want)
		}
	}
}
