package effect_test

import (
	"testing"

	"github.com/zadykian/ws/mac/internal/effect"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		name     string
		old, new string
		want     string
	}{
		{"equal", "a\nb\n", "a\nb\n", ""},
		{"empty", "", "", ""},
		{"added file", "", "a\n", "--- f\n+++ f\n@@ -0,0 +1 @@\n+a\n"},
		{"removed line", "a\nb\n", "a\n", "--- f\n+++ f\n@@ -1,2 +1 @@\n a\n-b\n"},
		{"changed line", "a\nb\nc\n", "a\nx\nc\n", "--- f\n+++ f\n@@ -1,3 +1,3 @@\n a\n-b\n+x\n c\n"},
		{"three lines of context", "1\n2\n3\n4\n5\n6\n7\n8\n9\n", "1\n2\n3\n4\nx\n6\n7\n8\n9\n",
			"--- f\n+++ f\n@@ -2,7 +2,7 @@\n 2\n 3\n 4\n-5\n+x\n 6\n 7\n 8\n"},
		{"changes six lines apart share a hunk", "1\n2\n3\n4\n5\n6\n7\n8\n", "x\n2\n3\n4\n5\n6\n7\ny\n",
			"--- f\n+++ f\n@@ -1,8 +1,8 @@\n-1\n+x\n 2\n 3\n 4\n 5\n 6\n 7\n-8\n+y\n"},
		{"changes seven lines apart do not", "1\n2\n3\n4\n5\n6\n7\n8\n9\n", "x\n2\n3\n4\n5\n6\n7\n8\ny\n",
			"--- f\n+++ f\n@@ -1,4 +1,4 @@\n-1\n+x\n 2\n 3\n 4\n@@ -6,4 +6,4 @@\n 6\n 7\n 8\n-9\n+y\n"},
		{"emptied file", "a\n", "", "--- f\n+++ f\n@@ -1 +0,0 @@\n-a\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := effect.Diff("f", []byte(tt.old), []byte(tt.new)); got != tt.want {
				t.Errorf("Diff = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDiffAs(t *testing.T) {
	got := effect.DiffAs("/dev/null", "f", nil, []byte("a\nb\n"))
	if want := "--- /dev/null\n+++ f\n@@ -0,0 +1,2 @@\n+a\n+b\n"; got != want {
		t.Errorf("DiffAs = %q, want %q", got, want)
	}
}
