package main

import (
	"maps"
	"testing"
)

// expect fails the test unless the mirror's lines that hold text are want's, at their origins.
func expect(t *testing.T, m *mirror, want map[string]origin) {
	got := map[string]origin{}
	for i, line := range m.lines {
		if line != "" {
			got[line] = m.from[i]
		}
	}
	if !maps.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestHashMirror(t *testing.T) {
	src := `#!/bin/sh
# Says why,
  # indented.
echo "# not a comment" # nor this
# shellcheck disable=SC2016 # the reason stays
# shellcheck source=x.sh
# yamllint finds the files, a sentence.
#   - a list
`
	expect(t, lineMirror([]byte(src), "#", hashDirective), map[string]origin{
		"Says why,": {2, 3}, "indented.": {3, 5}, "the reason stays": {5, 31},
		"yamllint finds the files, a sentence.": {7, 3}, "  - a list": {8, 3},
	})
}
