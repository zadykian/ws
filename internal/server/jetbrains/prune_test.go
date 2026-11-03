package jetbrains

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func notRunning(t *testing.T, m *effecttest.Machine, dir string) {
	m.Results["pgrep -f -- "+dir+"/"] = effecttest.Result{Err: exit(t, "1")}
}

func backendFixture(t *testing.T) (*effecttest.Machine, Backend) {
	m, s, _ := fixture(1 << 30)
	b, err := ParseReleases([]byte(list), []string{"GO"}, "linux", testDist)
	if err != nil {
		t.Fatal(err)
	}
	m.Pages[b[0].Link] = testBody
	return m, Backend{S: s, B: b[0]}
}

func TestOthersRemovesTheBuildsOfTheProductAlone(t *testing.T) {
	m, b := backendFixture(t)
	old := testDist + "/aaaaaaaaaaaaa_goland-2026.1.9"
	other := testDist + "/bbbbbbbbbbbbb_rider-2026.1.9"
	for _, d := range []string{b.B.Dir, old, other} {
		m.Dirs[d] = 0o755
	}
	notRunning(t, m, old)
	changes, err := others(context.Background(), b.S, b.B)
	if err != nil || len(changes) != 1 || !strings.Contains(changes[0].String(), old) {
		t.Fatalf("%v, %v", changes, err)
	}
}

func TestOthersKeepsABuildAProcessRunsFrom(t *testing.T) {
	m, b := backendFixture(t)
	old := testDist + "/aaaaaaaaaaaaa_goland-2026.1.9"
	m.Dirs[old] = 0o755
	m.Results["pgrep -f -- "+old+"/"] = effecttest.Result{Out: []byte("77\n")}
	changes, err := others(context.Background(), b.S, b.B)
	if err != nil || len(changes) != 0 {
		t.Fatalf("%v, %v", changes, err)
	}
	if out := b.S.Out.(*effecttest.Out); !slices.ContainsFunc(out.Lines, func(l string) bool { return strings.Contains(l, "stays") }) {
		t.Errorf("not said: %v", out.Lines)
	}
}

func TestOthersHasNothingWhereTheDirectoryIsNotMade(t *testing.T) {
	m, b := backendFixture(t)
	delete(m.Dirs, testDist)
	if changes, err := others(context.Background(), b.S, b.B); err != nil || len(changes) != 0 {
		t.Errorf("%v, %v", changes, err)
	}
}

func TestPruneRemovesUnderTheLockWhereTheBuildIsThere(t *testing.T) {
	m, s, _ := fixture(0)
	gone := testDist + "/aaaaaaaaaaaaa_goland-2026.1.9"
	m.Results["rm -rf -- "+gone] = effecttest.Result{}
	if err := (prune{s, gone}).Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(m.Calls, "run rm -rf -- "+gone) {
		t.Error("removed a build that another run moved or removed")
	}
	m.Dirs[gone] = 0o755
	if err := (prune{s, gone}).Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Calls, "run rm -rf -- "+gone) {
		t.Errorf("kept the build: %v", m.Calls)
	}
	m.Results["rm -rf -- "+gone] = effecttest.Result{Err: errors.New("busy")}
	if err := (prune{s, gone}).Apply(context.Background()); err == nil {
		t.Error("a failed removal is no error")
	}
}
