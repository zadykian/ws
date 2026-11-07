package zed

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/server/values"
)

// render returns the settings file for the home.
func render(t *testing.T, home string) []byte {
	v := values.Default()
	v.Home = home
	got, err := settingsFile(v)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestSettingsAreJSONUnderTheirCommentLine(t *testing.T) {
	comment, body, _ := strings.Cut(string(render(t, "/home/ws")), "\n")
	if !strings.HasPrefix(comment, "//") {
		t.Fatalf("no comment line first: %q", comment)
	}
	if !json.Valid([]byte(body)) {
		t.Fatalf("not JSON:\n%s", body)
	}
}

func TestSettingsNameTheTerminalWrapper(t *testing.T) {
	if got := string(render(t, "/root")); !strings.Contains(got, `"`+terminalPath+`"`) {
		t.Errorf("settings lack %q", terminalPath)
	}
}
