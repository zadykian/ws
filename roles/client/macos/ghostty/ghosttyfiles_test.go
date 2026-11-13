package ghostty

import (
	"github.com/zadykian/ws/engine/core/resource/file"
	"slices"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/userfile"
)

func fileChanges(t *testing.T, m *effecttest.Machine, holds bool) []step.Change {
	g := &Ghostty{M: m, Out: &effecttest.Out{}}
	dir := "/h/.config/ghostty"
	ws, err := userfile.File{M: m, Path: dir + "/ws.ghostty", Text: []byte("new\n"), Mode: 0o644}.Plan()
	if err != nil {
		t.Fatal(err)
	}
	var lines []step.Change
	if !holds {
		lines = []step.Change{file.UserWrite{Path: dir + "/config"}}
	}
	return g.fileChanges(ws, nil, userfile.DirLink{}, holds, lines, nil)
}

func TestGhosttyFileChangesWriteBothFiles(t *testing.T) {
	cs := fileChanges(t, effecttest.NewMachine(), false)
	if _, ok := cs[0].(file.UserWrite); len(cs) != 2 || !ok {
		t.Errorf("changes = %v; want two files", cs)
	}
}

func TestGhosttyFileChangesSetTheModeOfAFileThatMatches(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/h/.config/ghostty"] = 0o755
	m.Files["/h/.config/ghostty/ws.ghostty"] = []byte("new\n")
	m.Modes["/h/.config/ghostty/ws.ghostty"] = 0o600
	cs := fileChanges(t, m, true)
	if len(cs) != 1 || !slices.ContainsFunc(cs, func(c step.Change) bool { _, ok := c.(file.UserChmod); return ok }) {
		t.Errorf("changes = %v; want the mode alone", cs)
	}
}

func TestGhosttyKeepsTheOldConfigAsTheTimeStamped(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/.config/ghostty/config"] = []byte("theme = x\n")
	g := &Ghostty{M: m, Home: "/h", Now: func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }}
	cs, err := g.configLines("/h/.config/ghostty/config", "/h/.config/ghostty/ws.ghostty").Plan(t.Context())
	want := file.UserBackup{Path: "/h/.config/ghostty/config", To: "/h/.config/ghostty/config.20260102030405~"}
	if err != nil || len(cs) != 2 || cs[0] != step.Change(want) {
		t.Errorf("changes = %v, %v; want %v first", cs, err, want)
	}
}
