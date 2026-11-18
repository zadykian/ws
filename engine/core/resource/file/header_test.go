package file_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/resource/file"
)

func TestHeaderStartsWithTheSignAndTheMark(t *testing.T) {
	for _, sign := range []string{"#", "//"} {
		got := file.Header(sign)
		if !strings.HasPrefix(got, sign+" "+file.Mark+", ") || strings.Contains(got, "\n") {
			t.Errorf("Header(%q) = %q", sign, got)
		}
	}
}

func TestOursReadsTheFirstLine(t *testing.T) {
	for _, tt := range []struct {
		name string
		text string
		want bool
	}{
		{"the header", file.Header("#") + "\nkey = 1\n", true},
		{"no newline", file.Header("//"), true},
		{"an old Mac header", "# Written by another tool: its next run replaces changes made here.\n", false},
		{"the mark on line 2 alone", "#!/bin/sh\n" + file.Header("#") + "\n", false},
		{"empty", "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := file.Ours([]byte(tt.text)); got != tt.want {
				t.Errorf("Ours = %v, want %v", got, tt.want)
			}
		})
	}
}
