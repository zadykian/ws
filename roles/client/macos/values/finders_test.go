package values

import (
	"strings"
	"testing"
)

// The reader, HasLine, replace and Keep agree on which keys a text holds.
func TestFindersFindWhatEntriesReturns(t *testing.T) {
	texts := []string{
		"", "# WS_X: 1\n", "WS_USER: ada\nWS_SLOT: 3\n", "WS_USER: 'a' # me\n\n  # c\nWS_SLOT: \"1\"\n",
		"WS_USER:\nWS_SLOT: ~\nWS_REMOTE: null\n", "WS_USER: a\r\nWS_SLOT: 1\r\n", "WS_GIT_SIGNING_PUB_KEY: x\nWS_SLOT: 1\n",
	}
	keys := []string{"WS_USER", "WS_SLOT", "WS_REMOTE", "WS_X", "WS_GIT_SIGNING_PUB_KEY"}
	for _, text := range texts {
		got, errs := entries(text)
		if len(errs) > 0 {
			t.Fatalf("%q: %v", text, errs)
		}
		read := map[string]bool{}
		for _, e := range got {
			read[e.key] = true
		}
		lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
		for _, k := range keys {
			if HasLine(text, k) != read[k] {
				t.Errorf("%q: HasLine(%s) = %v; entries has it: %v", text, k, HasLine(text, k), read[k])
			}
			// replace sets a line it finds, and adds one it does not.
			want := strings.Count(text, "\n")
			if text != "" && !strings.HasSuffix(text, "\n") {
				want++
			}
			if !read[k] {
				want++
			}
			if got := strings.Count(replace(text, k, "v"), "\n"); got != want {
				t.Errorf("%q: replace(%s) = %q", text, k, replace(text, k, "v"))
			}
		}
		for _, e := range got {
			line := strings.TrimSuffix(lines[e.line-1], "\r") + "\n"
			if KeyScope(e.key).Keep(line) != line || (KeyScope(e.key) == MacFile) == (ServerFile.Keep(line) == line) {
				t.Errorf("%q: Keep does not place %s by its key", text, e.key)
			}
		}
	}
}
