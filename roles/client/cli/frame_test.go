package cli

import (
	"os"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// valuesCheckFile is the frame's file that runs the values check, in the user's home.
const valuesCheckFile = "~/Library/Caches/ws-values.checked"

// valuesCheckPart reads the frame's file of the values check from the Mac's root config.
func valuesCheckPart(t *testing.T) map[string]any {
	text, err := os.ReadFile(macTreeRoot + "/" + macRoleFile)
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Bootstrap struct {
			Files map[string]map[string]any `toml:"files"`
		} `toml:"bootstrap"`
	}
	if err := toml.Unmarshal(text, &c); err != nil {
		t.Fatal(err)
	}
	part, ok := c.Bootstrap.Files[valuesCheckFile]
	if !ok {
		t.Fatalf("%s holds no [bootstrap.files.%q]: the Mac's values go unchecked until a unit reads one", macRoleFile, valuesCheckFile)
	}
	return part
}

// TestValuesCheckFileIsTheUsersTemplate keeps the file that runs the check in the user's home with
// no owner or group, as mise reads and writes it as the user. A template renders before the
// packages' phase, where a unit would run.
func TestValuesCheckFileIsTheUsersTemplate(t *testing.T) {
	part := valuesCheckPart(t)
	if part["template"] != true {
		t.Errorf("template = %v, want true", part["template"])
	}
	for _, k := range []string{"owner", "group", "mode"} {
		if v, ok := part[k]; ok {
			t.Errorf("%s = %v: a file with no owner or group is the user's", k, v)
		}
	}
}
