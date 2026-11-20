package git

import (
	"strings"
	"testing"
)

// lookupOf is the run's env of the secrets task, which holds env alone.
func lookupOf(env map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	}
}

const oneToken = `["https://c.example"]
token = "` + dummy + `-c"
`

// In the task's run, the tokens are the input's, which mise writes to the file before any unit and
// in a plan nowhere. Without the input, they are the file's.
func TestRunTokensTakeTheInputOfTheRun(t *testing.T) {
	m := tokensMachine(twoTokens)
	for _, c := range []struct {
		lookup func(string) (string, bool)
		urls   []string
	}{
		{lookupOf(map[string]string{TokensVar: oneToken}), []string{"https://c.example"}},
		{lookupOf(map[string]string{TokensVar: ""}), nil},
		{lookupOf(map[string]string{}), []string{"https://a.example/x", "https://b.example"}},
		{nil, []string{"https://a.example/x", "https://b.example"}},
	} {
		tokens, present, err := RunTokens(m, c.lookup)
		if err != nil || present != (c.urls != nil) || len(tokens) != len(c.urls) {
			t.Errorf("want %q: %d tables, present %v, %v", c.urls, len(tokens), present, err)
		}
		for _, url := range c.urls {
			if _, ok := tokens[url]; !ok {
				t.Errorf("want %q: %s is missing", c.urls, url)
			}
		}
	}
}

// A refused input is named by its variable, never by its value.
func TestRunTokensRefuseTheInputByItsName(t *testing.T) {
	bad := lookupOf(map[string]string{TokensVar: `["https://a.example"]` + "\ntoken = \"" + dummy + " x\"\n"})
	_, _, err := RunTokens(tokensMachine(twoTokens), bad)
	if err == nil || !strings.Contains(err.Error(), TokensVar) || strings.Contains(err.Error(), dummy) {
		t.Errorf("got %v, want a refusal that names %s alone", err, TokensVar)
	}
}
