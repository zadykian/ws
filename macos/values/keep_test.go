package values

import (
	"strings"
	"testing"
)

// Keep places each key that entries reads by the file of its scope.
func TestKeepPlacesWhatEntriesReturns(t *testing.T) {
	texts := []string{
		"", "# WS_X: 1\n", "WS_TUNNEL_PORT: ada\nWS_SLOT: 3\n", "WS_TUNNEL_PORT: 'a' # me\n\n  # c\nWS_SLOT: \"1\"\n",
		"WS_TUNNEL_PORT:\nWS_SLOT: ~\nWS_REMOTE: null\n", "WS_TUNNEL_PORT: a\r\nWS_SLOT: 1\r\n", "WS_REMOTE: x\nWS_SLOT: 1\n",
	}
	for _, text := range texts {
		got, errs := entries(text)
		if len(errs) > 0 {
			t.Fatalf("%q: %v", text, errs)
		}
		lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
		for _, e := range got {
			line := strings.TrimSuffix(lines[e.line-1], "\r") + "\n"
			if KeyScope(e.key).Keep(line) != line || (KeyScope(e.key) == MacFile) == (ServerFile.Keep(line) == line) {
				t.Errorf("%q: Keep does not place %s by its key", text, e.key)
			}
		}
	}
}
