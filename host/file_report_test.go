package host_test

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/step"
)

func TestFileReport(t *testing.T) {
	const header, wanted = "# Written by github.com/zadykian/ws\n", "owner 0:0, mode 644"
	tests := []struct {
		name, before    string
		mode            fs.FileMode
		found, conflict string
		diff            bool
	}{
		{name: "missing", found: "missing"},
		{name: "as wanted", before: header + "new\n", found: wanted},
		{name: "by hand", before: header + "old\n", found: wanted + ", other content", diff: true,
			conflict: "differs from what ws writes: a change by hand, or another version of ws"},
		{name: "no line", before: "old\n", found: wanted + ", other content", diff: true,
			conflict: "holds no Written by line: ws did not write it"},
		{name: "mode", before: header + "new\n", mode: 0o600, found: "owner 0:0, mode 600",
			conflict: "its owner or mode changed"},
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
		if got.Found != tt.found || got.Conflict != tt.conflict {
			t.Errorf("%s: found %q, conflict %q, want %q, %q", tt.name, got.Found, got.Conflict, tt.found, tt.conflict)
		}
		if (got.Diff != "") != tt.diff || tt.diff && !strings.Contains(got.Diff, "+new") {
			t.Errorf("%s: diff = %q, want one: %v", tt.name, got.Diff, tt.diff)
		}
	}
}

func TestFileReportKeepsASecretOut(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/key"] = []byte("old-value\n")
	f := host.File{M: m, Path: "/key", Content: []byte("new-value\n"), Mode: 0o644, Secret: true}
	got := fmt.Sprintf("%+v", reportOf(t, f))
	if strings.Contains(got, "value") {
		t.Errorf("item = %s, want neither value", got)
	}
}
