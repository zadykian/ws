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
		"Package p is a test.": {1, 4}, "xxx says why.": {4, 4}, "\tcode stays code": {6, 3},
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

func TestMirrorOf(t *testing.T) {
	for file, src := range map[string]string{"a.json": "// a\n# b\n", "c.json.tmpl": " // a\n", "Makefile": "# a\n// b\n"} {
		if m, err := mirrorOf(file, []byte(src)); err != nil || m.texts != 1 {
			t.Errorf("%s: got %v, %v, want one comment", file, m, err)
		}
	}
}

func TestUnchecked(t *testing.T) {
	for file, want := range map[string]bool{
		"docs/a.md": true, ".vale/styles/a.yml": true, ".github/ISSUE_TEMPLATE/change.yml": true, "go.sum": true,
		"roles/server/ubuntu/signoz/deployment/a.yaml": true, "roles/server/ubuntu/signoz/casting.yaml.lock": true,
		"roles/server/ubuntu/signoz/casting.yaml": false, "x/LICENSE": true, "x/package-lock.json": true,
		"go.mod": false, ".github/lint-requirements.in": false, "a.mdx": false,
	} {
		if got := unchecked.MatchString(file); got != want {
			t.Errorf("%s: got %v, want %v", file, got, want)
		}
	}
}

func TestGoMirrorMasksDocNames(t *testing.T) {
	src := `package p

// Validate checks it.
func Validate() {}

// An Allocator splits it.
//
// Allocator stays a word past the first line.
type Allocator struct {
	// The Allocate field, of Allocator.
	Allocate int
}

// Validator is no name of this func.
func Valid() {}

// é is a name too.
var é = 1
`
	m, err := goMirror([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	expect(t, m, map[string]origin{
		"xxxxxxxx checks it.": {3, 4}, "An xxxxxxxxx splits it.": {6, 4},
		"Allocator stays a word past the first line.": {8, 4},
		"The xxxxxxxx field, of Allocator.":           {10, 5},
		"Validator is no name of this func.":          {14, 4}, "x is a name too.": {17, 4},
	})
}

func TestMirrorHidesNothing(t *testing.T) {
	m := newMirror()
	m.add("~~~ opens no fence", origin{1, 3})
	m.add("```go", origin{2, 3})
	m.add("<!-- vale off --> and <pre>", origin{3, 3})
	want := []string{"", "xxx opens no fence", "xxxgo", "‹!-- vale off --> and ‹pre>"}
	if !slices.Equal(m.lines, want) {
		t.Errorf("got %q, want %q", m.lines, want)
	}
}
