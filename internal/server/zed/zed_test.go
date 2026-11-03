package zed

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/server/values"
)

func TestSettingsAreValidJSONWithTheUsersPaths(t *testing.T) {
	v := values.Default()
	v.Home = "/home/ws"
	got, err := settingsFile(v)
	if err != nil {
		t.Fatal(err)
	}
	_, body, _ := strings.Cut(string(got), "\n")
	if !json.Valid([]byte(body)) {
		t.Fatalf("not JSON:\n%s", body)
	}
	for _, want := range []string{"/home/ws/go/bin/gopls", "/home/ws/.dotnet/tools/roslyn-language-server",
		"/home/ws/go/bin/dlv"} {
		if !strings.Contains(body, `"`+want+`"`) {
			t.Errorf("settings lack %q", want)
		}
	}
}
