package host_test

import (
	"io/fs"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/step"
)

func TestFileReport(t *testing.T) {
	const header, wanted = "# Written by github.com/zadykian/ws\n", "owner 0:0, mode 644"
	tests := []struct {
		name, before string
		mode         fs.FileMode
		found        string
	}{
		{name: "missing", found: "missing"},
		{name: "as wanted", before: header + "new\n", found: wanted},
		{name: "content", before: header + "old\n", found: wanted + ", other content"},
		{name: "mode", before: header + "new\n", mode: 0o600, found: "owner 0:0, mode 600"},
	}
	for _, tt := range tests {
		m := effecttest.NewMachine()
		if tt.before != "" {
			m.Files["/a"] = []byte(tt.before)
		}
		if tt.mode != 0 {
			m.Modes["/a"] = tt.mode
		}
		f := host.File{M: m, Path: "/a", Content: []byte(header + "new\n"), Mode: 0o644}
		got := reportOf(t, host.Validated{File: f})
		if got.Kind != step.FileItem || got.Name != "/a" || got.Want != wanted {
			t.Errorf("%s: item = %+v, want the file's wanted state", tt.name, got)
		}
		if got.Found != tt.found {
			t.Errorf("%s: found %q, want %q", tt.name, got.Found, tt.found)
		}
	}
}
