package host_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/host"
)

func TestRender(t *testing.T) {
	fsys := fstest.MapFS{
		"a.tmpl":   {Data: []byte("user {{.User}}\n")},
		"bad.tmpl": {Data: []byte("{{.Missing}}")},
		"syn.tmpl": {Data: []byte("{{")},
	}
	data := map[string]string{"User": "ws"}
	got, err := host.Render(fsys, "a.tmpl", data)
	if err != nil || string(got) != "user ws\n" {
		t.Errorf("Render = %q, %v; want user ws", got, err)
	}
	for name, want := range map[string]string{"bad.tmpl": "no entry for key", "syn.tmpl": "unclosed", "no.tmpl": "file does not exist"} {
		if _, err := host.Render(fsys, name, data); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Render %s: err = %v, want %q", name, err, want)
		}
	}
}
