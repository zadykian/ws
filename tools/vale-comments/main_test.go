package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	inDir(t, fakeVale, map[string]string{"a.sh": "# Says why.\n", "b.md": "# Title\n"})
	if out, err := exec.Command("git", "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != 1 || stdout.String() != "a.sh:1:5 X.Rule m.\n" {
		t.Errorf("got %d and %q, want 1 and a.sh's alert", code, &stdout)
	}
	if code := run([]string{"b.md"}, &stdout, &stderr); code != 0 {
		t.Errorf("Markdown alone: got %d, want 0", code)
	}
	if code := run([]string{"none"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "git lists no file") {
		t.Errorf("no file: got %d and %q, want 2 and git's listing named", code, &stderr)
	}
	valeCommand = []string{"sh", "-c", "exit 2", "vale"}
	if code := run(nil, &stdout, &stderr); code != 2 {
		t.Errorf("Vale fails: got %d, want 2", code)
	}
}
