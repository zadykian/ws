package session

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

// caseWords returns the words of the case table in the script's main.
func caseWords(script []byte) []string {
	_, body, _ := strings.Cut(string(script), "\nmain() {")
	var words []string
	for _, m := range regexp.MustCompile(`(?m)^    ([a-z]+)\) `).FindAllStringSubmatch(body, -1) {
		words = append(words, m[1])
	}
	slices.Sort(words)
	return words
}

// tableWords returns the words that the table sends to the files script, or to the remote script.
func tableWords(files bool) []string {
	var words []string
	for _, rule := range opRules {
		for _, snd := range rule.sends {
			if snd.files == files && !slices.Contains(words, snd.word) {
				words = append(words, snd.word)
			}
		}
	}
	slices.Sort(words)
	return words
}

func TestOpsMatchTheScripts(t *testing.T) {
	for name, tc := range map[string]struct {
		script []byte
		files  bool
	}{"remote": {RemoteScript, false}, "files": {RemoteFilesScript, true}} {
		if got, want := tableWords(tc.files), caseWords(tc.script); !slices.Equal(got, want) {
			t.Errorf("%s: the table sends %q; the script's case table holds %q", name, got, want)
		}
	}
}
