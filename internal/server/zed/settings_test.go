package zed

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	editorzed "github.com/zadykian/ws/internal/editor/zed"
	"github.com/zadykian/ws/internal/server/devtools"
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

func TestTheTemplateEqualsTheAdaptersRenderAtEachHome(t *testing.T) {
	for _, home := range []string{"/home/ws", "/root"} {
		want, err := editorzed.ServerSettings(devtools.ServerPaths(home), devtools.Dlv(home), terminalPath)
		if err != nil {
			t.Fatal(err)
		}
		if got := render(t, home); !bytes.Equal(got, want) {
			t.Errorf("%s: %d bytes, want %d\n%s", home, len(got), len(want), got)
		}
	}
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
