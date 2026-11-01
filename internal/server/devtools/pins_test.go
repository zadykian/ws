package devtools

import (
	"os"
	"strings"
	"testing"
)

// TestGoVersionAgreesWithGroupVars checks the copy of the pin that the helix role still reads.
func TestGoVersionAgreesWithGroupVars(t *testing.T) {
	data, err := os.ReadFile("../../../group_vars/all.yml")
	if err != nil {
		t.Fatal(err)
	}
	want := "devtools_go_version: \"" + GoVersion + "\"\n"
	if !strings.Contains(string(data), want) {
		t.Errorf("group_vars/all.yml lacks %q", want)
	}
}
