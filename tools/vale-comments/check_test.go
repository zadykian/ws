package main

import (
	"os"
	"slices"
	"testing"
)

// fakeVale alerts on line 2 of the last mirror it gets, and exits with 1, as Vale does then.
const fakeVale = `eval "last=\${$#}"
printf '{"%s":[{"Check":"X.Rule","Message":"m.","Line":2,"Span":[3,5]}]}' "$last"
exit 1`

// inDir writes the files in a new directory, and runs the test there with script as Vale.
func inDir(t *testing.T, script string, files map[string]string) {
	t.Chdir(t.TempDir())
	for name, src := range files {
		if err := os.WriteFile(name, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	valeCommand = []string{"sh", "-c", script, "vale"}
	t.Cleanup(func() { valeCommand = []string{"tools/run", "vale"} })
}

func TestCheck(t *testing.T) {
	inDir(t, fakeVale, map[string]string{"a.sh": "x=1\n  # Says why.\n", "b.md": "# Title\n"})
	found, err := check([]string{"a.sh", "b.md"}, os.Stderr)
	if want := []finding{{"a.sh", "X.Rule", "m.", 2, 7}}; err != nil || !slices.Equal(found, want) {
		t.Errorf("got %v, %v, want %v", found, err, want)
	}
}
