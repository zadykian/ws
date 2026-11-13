package file_test

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
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
		f := file.File{M: m, Path: "/a", Content: []byte(header + "new\n"), Mode: 0o644}
		got := reportOf(t, file.Validated{File: f})
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
	f := file.File{M: m, Path: "/key", Content: []byte("new-value\n"), Mode: 0o644, Secret: true}
	got := fmt.Sprintf("%+v", reportOf(t, f))
	if strings.Contains(got, "value") {
		t.Errorf("item = %s, want neither value", got)
	}
}

// built is a part that builds its resource as it plans, as a role's wrapper does.
type built struct{ r resource.Resource }

func (b built) Plan(ctx context.Context) ([]resource.Change, error) { return b.r.Plan(ctx) }

func TestAResourceBuiltInPlanStandsAsAnItem(t *testing.T) {
	const written = "differs from what ws writes: a change by hand, or another version of ws"
	m := effecttest.NewMachine()
	m.Files["/a"] = []byte("old\n")
	f := file.File{M: m, Path: "/a", Content: []byte("new\n"), Mode: 0o644}
	secret := f
	secret.Secret = true
	mode := f
	mode.Content, mode.Mode = []byte("old\n"), 0o600
	tests := []struct {
		name     string
		r        resource.Resource
		conflict string
		diff     bool
	}{
		{"file", f, written, true},
		{"checked", file.Validated{File: f}, written, true},
		{"secret", secret, written, false},
		{"line", file.Line{M: m, Path: "/a", Text: "new", Mode: 0o644}, "a line differs from what ws sets", true},
		{"mode", mode, "its owner or mode changed", false},
	}
	for _, tt := range tests {
		got, err := step.Read(t.Context(), []step.Step{resource.Group{Title: "T", Parts: []resource.Resource{built{tt.r}}}})
		if err != nil || len(got[0].Items) != 1 {
			t.Fatalf("%s: reports = %+v, %v", tt.name, got, err)
		}
		it := got[0].Items[0]
		if it.Name != "/a" || it.Conflict != tt.conflict || (it.Diff != "") != tt.diff {
			t.Errorf("%s: item = %+v, want conflict %q, a diff: %v", tt.name, it, tt.conflict, tt.diff)
		}
	}
}

func TestAMissingFileBuiltInPlanStandsAgainstNothing(t *testing.T) {
	f := file.File{M: effecttest.NewMachine(), Path: "/a", Content: []byte("new\n"), Mode: 0o644}
	got, err := step.Read(t.Context(), []step.Step{resource.Group{Title: "T", Parts: []resource.Resource{built{f}}}})
	if err != nil || len(got[0].Items) != 1 || got[0].Items[0].Found != "missing" || got[0].Items[0].Conflict != "" {
		t.Errorf("reports = %+v, %v", got, err)
	}
}

func TestAFileInAGroupStandsOnce(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/a"] = []byte("old\n")
	f := file.File{M: m, Path: "/a", Content: []byte("new\n"), Mode: 0o644}
	got, err := step.Read(t.Context(), []step.Step{resource.Group{Title: "T", Parts: []resource.Resource{f}}})
	if err != nil || len(got[0].Items) != 1 {
		t.Errorf("reports = %+v, %v", got, err)
	}
}
