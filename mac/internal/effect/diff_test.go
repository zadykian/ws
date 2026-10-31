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
		{"added file", "", "a\n", "--- f\n+++ f\n@@ -1,0 +1,1 @@\n+a\n"},
		{"removed line", "a\nb\n", "a\n", "--- f\n+++ f\n@@ -1,2 +1,1 @@\n a\n-b\n"},
		{"changed line", "a\nb\nc\n", "a\nx\nc\n", "--- f\n+++ f\n@@ -1,3 +1,3 @@\n a\n-b\n+x\n c\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := effect.Diff("f", []byte(tt.old), []byte(tt.new)); got != tt.want {
				t.Errorf("Diff = %q, want %q", got, tt.want)
			}
		})
	}
}
