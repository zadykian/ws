package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOpenLogIsPrivateUnderTheStateDirectory(t *testing.T) {
	state := t.TempDir()
	p := withEnv(fakeProcess(), map[string]string{"XDG_STATE_HOME": state})
	f, path := openLog(p, "ws server apply", time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC))
	if f == nil {
		t.Fatal("no log")
	}
	defer func() { _ = f.Close() }()
	if want := filepath.Join(state, "ws", "ws-server-apply-20261105T120000Z.log"); path != want {
		t.Errorf("path %q, want %q", path, want)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("mode of %s: %v, %v", path, info, err)
	}
}

func TestOpenLogFallsBackToHomeAndNeedsAnAbsoluteDirectory(t *testing.T) {
	home := t.TempDir()
	f, path := openLog(withEnv(fakeProcess(), map[string]string{"HOME": home}), "ws mac apply", time.Now())
	if f == nil || !strings.HasPrefix(path, filepath.Join(home, ".local", "state", "ws")) {
		t.Fatalf("log %v at %q", f, path)
	}
	_ = f.Close()
	for _, env := range []map[string]string{{}, {"XDG_STATE_HOME": "state"}} {
		if f, path := openLog(withEnv(fakeProcess(), env), "ws mac apply", time.Now()); f != nil || path != "" {
			t.Errorf("env %v made %q", env, path)
		}
	}
}
