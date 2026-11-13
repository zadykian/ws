package change_test

import (
	"testing"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
)

// ws replaces its own files whole, so what one held stands in the excerpt.
func TestAnExcerptKeepsTheDiffOfAFileWsReplaces(t *testing.T) {
	w := change.Write{Path: "/f", Before: []byte("old\n"), After: []byte("new\n"), Exists: true}
	if got := step.ExcerptOf(w); got != w.String() {
		t.Errorf("excerpt = %q, want the diff %q", got, w.String())
	}
	w = change.Write{Path: "/f", After: []byte("new\n"), Exists: true}
	if got := step.ExcerptOf(w); got != w.String() {
		t.Errorf("empty file: excerpt = %q, want the diff %q", got, w.String())
	}
}
