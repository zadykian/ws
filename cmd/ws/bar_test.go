package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestServerBarRefuses(t *testing.T) {
	file := filepath.Join(t.TempDir(), "no", "line")
	status, _, stderr := runWs(fakeProcess(), "", "server", "bar", file)
	if status != 1 || !strings.HasPrefix(stderr, "ws server bar: ") || !strings.Contains(stderr, "line.new") {
		t.Errorf("a directory that does not exist: status = %d, stderr = %q", status, stderr)
	}
	if status, _, _ = runWs(fakeProcess(), "", "server", "bar", "--interval", "500ms", file); status != 2 {
		t.Errorf("an interval under 1s: status = %d, want 2", status)
	}
	if status, _, _ = runWs(fakeProcess(), "", "server", "bar", "a", "b"); status != 2 {
		t.Errorf("two files: status = %d, want 2", status)
	}
}
