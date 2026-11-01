package jetbrains

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const sumLink = "https://download.jetbrains.com/go/goland-2026.2.3.tar.gz.sha256"

func TestBackendInstallsAMissingBuild(t *testing.T) {
	m, b := backendFixture(t)
	m.Results["curl -fsSL "+sumLink] = effecttest.Result{Out: []byte(strings.Repeat("ab", 32) + " goland.tar.gz\n")}
	notRunning(t, m, b.B.Dir)
	changes, err := b.Plan(context.Background())
	if err != nil || len(changes) != 1 || !strings.Contains(changes[0].String(), "would be installed in "+b.B.Dir) {
		t.Fatalf("%v, %v", changes, err)
	}
}

func TestBackendPlansNothingForABuildInPlace(t *testing.T) {
	m, b := backendFixture(t)
	m.Files[b.B.Dir+"/"+marker] = nil
	changes, err := b.Plan(context.Background())
	if err != nil || len(changes) != 0 {
		t.Errorf("%v, %v", changes, err)
	}
}

func TestBackendStopsWhereAProcessRunsFromAnUnmarkedBuild(t *testing.T) {
	m, b := backendFixture(t)
	m.Results["curl -fsSL "+sumLink] = effecttest.Result{Out: []byte(strings.Repeat("ab", 32))}
	m.Results["pgrep -f -- "+b.B.Dir+"/"] = effecttest.Result{Out: []byte("41\n42\n")}
	_, err := b.Plan(context.Background())
	if err == nil || !strings.Contains(err.Error(), "process 41, 42 runs from it") {
		t.Errorf("got %v", err)
	}
}

func TestBackendStopsOnAPageWithNoSum(t *testing.T) {
	m, b := backendFixture(t)
	m.Results["curl -fsSL "+sumLink] = effecttest.Result{Out: []byte("<html>")}
	if _, err := b.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "holds no SHA-256") {
		t.Errorf("got %v", err)
	}
}

func TestBackendKeepsOtherBuildsWithoutPrune(t *testing.T) {
	m, b := backendFixture(t)
	m.Files[b.B.Dir+"/"+marker] = nil
	m.Dirs[testDist+"/aaaaaaaaaaaaa_goland-2026.1.9"] = 0o755
	b.S.Prune = false
	if changes, err := b.Plan(context.Background()); err != nil || len(changes) != 0 {
		t.Errorf("%v, %v", changes, err)
	}
}
