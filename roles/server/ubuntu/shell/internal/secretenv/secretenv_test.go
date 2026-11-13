package secretenv

import (
	"os"
	"strings"
	"testing"
)

func TestQuoteIsShlexQuote(t *testing.T) {
	for in, want := range map[string]string{
		"": "''", "plain": "plain", "a b$x": "'a b$x'", "it's": `'it'"'"'s'`,
		"a@%+=:,./-_9": "a@%+=:,./-_9", "line\nbreak": "'line\nbreak'", "ü": "'ü'",
	} {
		if got := quote(in); got != want {
			t.Errorf("quote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderEnvIsTheRolesFileInOrder(t *testing.T) {
	got := renderEnv(map[string]string{"B": "it's", "A": "plain", "C": "", "D": "a b$x"})
	want, err := os.ReadFile("testdata/secret_env")
	if err != nil {
		t.Fatal(err)
	}
	// The role wrote the names in the order of the YAML, here B, A, C, D; ws sorts them.
	lines := strings.SplitAfter(string(want), "\n")[2:]
	sorted := []string{lines[1], lines[0], lines[2], lines[3]}
	if exp := envHeader + strings.Join(sorted, ""); string(got) != exp {
		t.Errorf("got %q, want %q", got, exp)
	}
}

func TestCheckEnvNamesNamesNoValue(t *testing.T) {
	if err := checkEnvNames(map[string]string{"OK_1": "v", "_X": "v"}); err != nil {
		t.Error(err)
	}
	err := checkEnvNames(map[string]string{"a-b": "hunter2", "1A": "hunter3", "A\n": "v"})
	if err == nil || strings.Contains(err.Error(), "hunter") || !strings.Contains(err.Error(), `"1A", "A\n", "a-b"`) {
		t.Errorf("got %v", err)
	}
}
