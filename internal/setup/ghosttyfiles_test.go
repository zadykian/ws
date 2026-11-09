package setup

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

func fileChanges(t *testing.T, m *effecttest.Machine, out *effecttest.Out, text string, has bool) []step.Change {
	g := &Ghostty{M: m, Out: out}
	dir := "/h/.config/ghostty"
	ws, err := userFile{M: m, Path: dir + "/ws.ghostty", Text: []byte("new\n"), Mode: 0o644}.plan()
	if err != nil {
		t.Fatal(err)
	}
	return g.fileChanges(dir+"/config", []byte(text), dir+"/ws.ghostty", ws, has)
}

func TestGhosttyFileChangesMakeTheDirectoryAndBothFiles(t *testing.T) {
	cs := fileChanges(t, effecttest.NewMachine(), &effecttest.Out{}, "", false)
	if len(cs) != 3 || !strings.HasPrefix(cs[0].String(), "would make /h/.config/ghostty") {
		t.Errorf("changes = %v; want a directory and two files", cs)
	}
}

func TestGhosttyFileChangesSetTheModeOfAFileThatMatches(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/h/.config/ghostty"] = 0o755
	m.Files["/h/.config/ghostty/ws.ghostty"] = []byte("new\n")
	m.Modes["/h/.config/ghostty/ws.ghostty"] = 0o600
	cs := fileChanges(t, m, &effecttest.Out{}, "", true)
	if len(cs) != 1 || !slices.ContainsFunc(cs, func(c step.Change) bool { _, ok := c.(SetMode); return ok }) {
		t.Errorf("changes = %v; want the mode alone", cs)
	}
}
