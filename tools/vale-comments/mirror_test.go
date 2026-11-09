package main

import (
	"maps"
	"slices"
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

func TestGoMirror(t *testing.T) {
	src := `// Package p is a test.
package p

// doc says why.
//
//	code stays code
//
//go:embed x
var doc string

var n = 1 //nolint:gosec // the reason stays
var é = "// not a comment" // trailing

/* a block
of two lines */
`
	m, err := goMirror([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	expect(t, m, map[string]origin{
		"Package p is a test.": {1, 4}, "doc says why.": {4, 4}, "\tcode stays code": {6, 3},
		"the reason stays": {11, 29}, "trailing": {12, 31},
	})
	// Two comments are two paragraphs, though on lines next to each other.
	if i := slices.Index(m.lines, "trailing"); m.lines[i-1] != "" {
		t.Errorf("the trailing comment joins the line before: %q", m.lines)
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
