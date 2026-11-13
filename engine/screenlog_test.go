package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// callIn returns the call of the command at path, in an environment of env.
func callIn(env map[string]string, path string) Call {
	return Call{Path: path, Getenv: func(k string) string { return env[k] }}
}

func TestOpenLogIsPrivateUnderTheStateDirectory(t *testing.T) {
	state := t.TempDir()
	f, path := openLog(callIn(map[string]string{"XDG_STATE_HOME": state}, "ws server ubuntu apply"), time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC))
	if f == nil {
		t.Fatal("no log")
	}
	defer func() { _ = f.Close() }()
	if want := filepath.Join(state, "ws", "ws-server-ubuntu-apply-20261105T120000Z.log"); path != want {
		t.Errorf("path %q, want %q", path, want)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("mode of %s: %v, %v", path, info, err)
	}
}

func TestOpenLogFallsBackToHomeAndNeedsAnAbsoluteDirectory(t *testing.T) {
	home := t.TempDir()
	f, path := openLog(callIn(map[string]string{"HOME": home}, "ws client macos apply"), time.Now())
	if f == nil || !strings.HasPrefix(path, filepath.Join(home, ".local", "state", "ws")) {
		t.Fatalf("log %v at %q", f, path)
	}
	_ = f.Close()
	for _, env := range []map[string]string{{}, {"XDG_STATE_HOME": "state"}} {
		if f, path := openLog(callIn(env, "ws client macos apply"), time.Now()); f != nil || path != "" {
			t.Errorf("env %v made %q", env, path)
		}
	}
}

func TestOpenLogKeepsTheNewest(t *testing.T) {
	state := t.TempDir()
	dir := filepath.Join(state, "ws")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	start := time.Now().Add(-time.Hour)
	for i := range logsKept + 5 {
		old := filepath.Join(dir, string(rune('a'+i))+".log")
		if err := os.WriteFile(old, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		at := start.Add(time.Duration(i) * time.Second)
		if err := os.Chtimes(old, at, at); err != nil {
			t.Fatal(err)
		}
	}
	f, _ := openLog(callIn(map[string]string{"XDG_STATE_HOME": state}, "ws client macos apply"), time.Now())
	_ = f.Close()
	left, _ := filepath.Glob(filepath.Join(dir, "*.log"))
	if len(left) != logsKept {
		t.Errorf("%d logs left, want %d", len(left), logsKept)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.log")); err == nil {
		t.Error("the oldest log stayed")
	}
}
