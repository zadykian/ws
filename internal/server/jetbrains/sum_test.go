package jetbrains

import (
	"strings"
	"testing"
)

func TestParseSum(t *testing.T) {
	sum := strings.Repeat("ab", 32)
	got, err := ParseSum([]byte(sum+" *goland-2026.2.3.tar.gz\n"), "https://x/sha256")
	if err != nil || got != sum {
		t.Errorf("got %q, %v", got, err)
	}
	for name, in := range map[string]string{
		"empty": "", "short": "abc goland.tar.gz", "upper": strings.ToUpper(sum), "html": "<html>",
	} {
		if _, err := ParseSum([]byte(in), "https://x/sha256"); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
