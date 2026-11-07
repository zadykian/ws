package fresh

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/devtools"
	"github.com/zadykian/ws/mac"
)

// The two editors take the same choices in their own files. These tests read both files as they
// are, so that a change to one that the other lacks fails here.

func at(v any, path ...string) any {
	for _, key := range path {
		m, _ := v.(map[string]any)
		v = m[key]
	}
	return v
}

func sharedFiles(t *testing.T) (fresh, zed, macZed map[string]any) {
	zedText, err := host.Render(os.DirFS("../zed"), "files/settings.json.tmpl", struct{ Home string }{"/home/ws"})
	if err != nil {
		t.Fatal(err)
	}
	fresh, _ = load(t)
	for data, out := range map[string]*map[string]any{string(zedText): &zed, string(mac.ZedSettings): &macZed} {
		if err := json.Unmarshal(withoutComments([]byte(data)), out); err != nil {
			t.Fatal(err)
		}
	}
	return fresh, zed, macZed
}

func TestTheEditorsShareTheirChoices(t *testing.T) {
	fresh, zed, macZed := sharedFiles(t)
	if a, b := at(fresh, "theme"), at(macZed, "theme", "dark"); a != b {
		t.Errorf("theme %v, Zed's %v", a, b)
	}
	if a, b := at(fresh, "editor", "rulers"), at(macZed, "wrap_guides"); !reflect.DeepEqual(a, b) {
		t.Errorf("rulers %v, Zed's wrap guides %v", a, b)
	}
	secs, _ := at(fresh, "editor", "auto_save_interval_secs").(float64)
	if ms := at(macZed, "autosave", "after_delay", "milliseconds"); secs*1000 != ms {
		t.Errorf("auto-save every %v seconds, Zed's %v ms", secs, ms)
	}
	for _, key := range []string{
		"whitespace_show", "whitespace_spaces_leading", "whitespace_spaces_inner",
		"whitespace_spaces_trailing", "whitespace_tabs_inner", "whitespace_tabs_trailing",
	} {
		if got, want := at(fresh, "editor", key), at(macZed, "show_whitespaces") == "all"; got != want {
			t.Errorf("%s %v, Zed's show_whitespaces %v", key, got, at(macZed, "show_whitespaces"))
		}
	}
	if got, want := at(fresh, "editor", "auto_save_enabled"), at(macZed, "autosave") != nil; got != want {
		t.Errorf("auto_save_enabled %v, Zed's autosave %v", got, at(macZed, "autosave"))
	}
	if at(fresh, "editor", "rainbow_brackets") != false {
		t.Error("rainbow_brackets is on; Zed's file doesn't colour brackets")
	}
	zedArgs := map[string]any{}
	for _, server := range at(zed, "lsp").(map[string]any) {
		zedArgs[at(server, "binary", "path").(string)] = at(server, "binary", "arguments")
	}
	shared := 0
	for _, list := range at(fresh, "lsp").(map[string]any) {
		for _, entry := range list.([]any) {
			if want, both := zedArgs[at(entry, "command").(string)]; both {
				shared++
				if got := at(entry, "args"); !reflect.DeepEqual(got, want) {
					t.Errorf("%v: args %v, Zed's %v", at(entry, "command"), got, want)
				}
			}
		}
	}
	if shared == 0 {
		t.Error("no server is named by both files")
	}
}

func TestEachServerIsNamedInBothFiles(t *testing.T) {
	fresh, zed, _ := sharedFiles(t)
	for name, path := range devtools.ServerPaths("/home/ws") {
		for editor, file := range map[string]map[string]any{"Zed": zed, "Fresh": fresh} {
			text, _ := json.Marshal(file)
			if editor == "Fresh" && name == "ansible-language-server" {
				continue // Fresh has no ansible server.
			}
			if !strings.Contains(string(text), path) {
				t.Errorf("%s's file lacks %s at %s", editor, name, path)
			}
		}
	}
}
