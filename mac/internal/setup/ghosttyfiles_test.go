package setup

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/step"
)

func fileChanges(m *effecttest.Machine, out *effecttest.Out, text string, has bool) []step.Change {
	g := &Ghostty{M: m, Out: out}
	dir := "/h/.config/ghostty"
	var old []byte
	if b, ok := m.Files[dir+"/ws.ghostty"]; ok {
		old = b
	}
	return g.fileChanges(dir+"/config", []byte(text), dir+"/ws.ghostty", old, []byte("new\n"), has)
}

func TestGhosttyFileChangesMakeTheDirectoryAndBothFiles(t *testing.T) {
	cs := fileChanges(effecttest.NewMachine(), &effecttest.Out{}, "", false)
	if len(cs) != 3 || !strings.HasPrefix(cs[0].String(), "would make /h/.config/ghostty") {
		t.Errorf("changes = %v; want a directory and two files", cs)
	}
}

func TestGhosttyFileChangesSetTheModeOfAFileThatMatches(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/h/.config/ghostty"] = 0o755
	m.Files["/h/.config/ghostty/ws.ghostty"] = []byte("new\n")
	m.Modes["/h/.config/ghostty/ws.ghostty"] = 0o600
	cs := fileChanges(m, &effecttest.Out{}, "", true)
	if len(cs) != 1 || !slices.ContainsFunc(cs, func(c step.Change) bool { _, ok := c.(SetMode); return ok }) {
		t.Errorf("changes = %v; want the mode alone", cs)
	}
}
