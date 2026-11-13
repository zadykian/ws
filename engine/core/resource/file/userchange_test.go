package file_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
)

func TestAFileChangeNamesItsFileInAReport(t *testing.T) {
	tests := []struct {
		change step.Change
		want   string
	}{
		{file.UserWrite{Path: "/f", After: []byte("a\n")}, "would create /f"},
		{file.UserWrite{Path: "/f", Before: []byte("b\n"), After: []byte("a\n"), Exists: true}, "would write /f"},
	}
	for _, tt := range tests {
		got, _, _ := strings.Cut(step.PendingOf(tt.change).Text, "\n")
		if got != tt.want {
			t.Errorf("%T: first line = %q, want %q", tt.change, got, tt.want)
		}
		if !strings.Contains(step.PendingOf(tt.change).Text, tt.change.String()) {
			t.Errorf("%T: the report lost the diff", tt.change)
		}
	}
}

// ws replaces its own files whole, so what one held stands in the excerpt.
func TestAnExcerptKeepsTheDiffOfAFileWsReplaces(t *testing.T) {
	w := file.UserWrite{Path: "/f", Before: []byte("old\n"), After: []byte("new\n"), Exists: true}
	if got := step.ExcerptOf(w); got != w.String() {
		t.Errorf("excerpt = %q, want the diff %q", got, w.String())
	}
	w = file.UserWrite{Path: "/f", After: []byte("new\n"), Exists: true}
	if got := step.ExcerptOf(w); got != w.String() {
		t.Errorf("empty file: excerpt = %q, want the diff %q", got, w.String())
	}
}

func TestAUserChangeSaysItsKind(t *testing.T) {
	for _, c := range []step.Change{file.UserWrite{Path: "/f"}, file.UserMkdir{Path: "/d"}, file.UserChmod{Path: "/f"}} {
		if got := step.KindOf(c); got != step.Write {
			t.Errorf("%T: kind = %s, want %s", c, got, step.Write)
		}
	}
}
