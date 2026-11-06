package fresh

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/zadykian/ws/host"
	editorfresh "github.com/zadykian/ws/internal/editor/fresh"
	"github.com/zadykian/ws/internal/server/devtools"
)

// parsed returns the file's header line and its JSON.
func parsed(t *testing.T, data []byte) (string, map[string]any) {
	header, body, _ := strings.Cut(string(data), "\n")
	var out map[string]any
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatal(err)
	}
	return header, out
}

func TestTheTemplateRendersWhatTheAdapterBuilt(t *testing.T) {
	for _, home := range []string{"/home/ws", "/root"} {
		rendered, err := host.Render(templates, "files/config.json.tmpl", struct{ Home string }{home})
		if err != nil {
			t.Fatal(err)
		}
		built, err := editorfresh.Config(devtools.ServerPaths(home))
		if err != nil {
			t.Fatal(err)
		}
		gotHeader, got := parsed(t, rendered)
		wantHeader, want := parsed(t, built)
		if gotHeader != wantHeader {
			t.Errorf("%s: header %q, want %q", home, gotHeader, wantHeader)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: the render differs from the adapter's config", home)
		}
	}
}
