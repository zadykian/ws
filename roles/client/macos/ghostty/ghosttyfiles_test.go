package ghostty

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
)

func fileChanges(t *testing.T, m *effecttest.Machine, holds bool) []step.Change {
	g := &Ghostty{M: m, Out: &effecttest.Out{}, Settings: []byte("new\n")}
	var settings []step.Change
	for _, part := range g.settings() {
		var err error
		if settings, err = resource.Append(context.Background(), settings, part); err != nil {
			t.Fatal(err)
		}
	}
	var lines []step.Change
	if !holds {
		lines = []step.Change{file.UserWrite{Path: "/h/.config/ghostty/config"}}
	}
	return g.fileChanges(settings, nil, file.UserDirLink{}, holds, lines, nil)
}

func TestGhosttyFileChangesMakeRootsDirectoryThenWriteBothFiles(t *testing.T) {
	cs := fileChanges(t, effecttest.NewMachine(), false)
	heads := []string{}
	for _, c := range cs {
		heads = append(heads, strings.SplitN(step.Headline(c), "\n", 2)[0])
	}
	want := []string{"create directory /etc/ws, owner 0:0, mode 755", "create /etc/ws/ghostty"}
	if len(heads) != 3 || !slices.Equal(heads[:2], want) {
		t.Errorf("headlines %q; want %q", heads, want)
	}
}

func TestGhosttyFileChangesPlanNothingForAFileThatMatches(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/etc/ws"] = 0o755
	g := &Ghostty{M: m, Out: &effecttest.Out{}, Settings: []byte("new\n")}
	m.Files["/etc/ws/ghostty"] = g.want()
	if cs := fileChanges(t, m, true); len(cs) != 0 {
		t.Errorf("changes = %v; want none", cs)
	}
}

func TestGhosttyKeepsTheOldConfigAsTheTimeStamped(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/.config/ghostty/config"] = []byte("theme = x\n")
	out := &effecttest.Out{}
	g := &Ghostty{M: m, Out: out, Home: "/h", Now: func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }}
	cs, err := g.configLines("/h/.config/ghostty/config").Plan(t.Context())
	want := file.UserBackup{M: m, Out: out, Path: "/h/.config/ghostty/config", To: "/h/.config/ghostty/config.20260102030405~"}
	if err != nil || len(cs) != 2 || cs[0] != step.Change(want) {
		t.Errorf("changes = %v, %v; want %v first", cs, err, want)
	}
}
