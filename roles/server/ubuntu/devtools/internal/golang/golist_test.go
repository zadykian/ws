package golang

import (
	"strings"
	"testing"
)

func TestGoSum(t *testing.T) {
	sum := strings.Repeat("ab", 32)
	list := `[{"version":"go1.27.3","files":[
		{"filename":"go1.27.3.src.tar.gz","sha256":"` + strings.Repeat("cd", 32) + `"},
		{"filename":"go1.27.3.linux-amd64.tar.gz","sha256":"` + sum + `"}]},
		{"version":"go1.27.2","files":[{"filename":"go1.27.2.linux-amd64.tar.gz","sha256":"` + sum + `"}]}]`
	if got, err := goSum([]byte(list), "go1.27.3.linux-amd64.tar.gz"); err != nil || got != sum {
		t.Errorf("got %q, %v", got, err)
	}
	tests := []struct{ name, list, file string }{
		{"a file the list lacks", list, "go1.27.3.linux-arm64.tar.gz"},
		{"a sum that is no SHA-256", `[{"files":[{"filename":"f","sha256":"ABC"}]}]`, "f"},
		{"no sum", `[{"files":[{"filename":"f"}]}]`, "f"},
		{"no releases", `[]`, "f"},
		{"no JSON", `<html>`, "f"},
	}
	for _, tt := range tests {
		if got, err := goSum([]byte(tt.list), tt.file); err == nil {
			t.Errorf("%s: got %q, want an error", tt.name, got)
		}
	}
}
