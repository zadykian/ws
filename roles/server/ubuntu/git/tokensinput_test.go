package git

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/roles/server/values"
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
	m, tk := tokenFixture(twoTokens)
	if _, err := Check(t.Context(), m, tk.A, values.Values{}, bad); err == nil || strings.Contains(err.Error(), dummy) {
		t.Errorf("the check of a refused input = %v", err)
	}
	tk.Lookup = bad
	if _, err := tk.Plan(t.Context()); err == nil {
		t.Error("the tokens planned a refused input")
	}
}

// A plan in the task's run shows the change of tokens.gitconfig that the run makes, and changes
// nothing. With the file's own tokens, or without the input, it shows none.
func TestTokensPlanTheInputOfTheRun(t *testing.T) {
	m, tk := tokenFixture(twoTokens)
	applyTokens(t, tk)
	before := string(m.Files[tokensPath])
	for _, c := range []struct {
		env  map[string]string
		want int
	}{
		{map[string]string{TokensVar: oneToken}, 1},
		{map[string]string{TokensVar: ""}, 1},
		{map[string]string{TokensVar: twoTokens}, 0},
		{map[string]string{}, 0},
	} {
		tk.Lookup = lookupOf(c.env)
		changes, err := tk.Plan(t.Context())
		if err != nil || len(changes) != c.want {
			t.Errorf("env %d: plan = %d changes, %v; want %d", len(c.env), len(changes), err, c.want)
		}
		for _, ch := range changes {
			if strings.Contains(ch.String(), dummy) {
				t.Errorf("the plan shows a token: %q", ch.String())
			}
		}
	}
	if string(m.Files[tokensPath]) != before {
		t.Error("a plan changed tokens.gitconfig")
	}
}
