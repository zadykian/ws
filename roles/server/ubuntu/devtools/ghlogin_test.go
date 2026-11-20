package devtools

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// lookupOf is the run's env of the secrets task, which holds env alone.
func lookupOf(env map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	}
}

// tokenText is TokenFile as mise writes it: ws's header, a comment, and the token on the last line.
const tokenText = "# Written by github.com/zadykian/ws, whose next run replaces any change made here.\n" +
	"# From WS_GITHUB_TOKEN.\ndummy-a\n"

// In the task's run, the token is the input's, which mise writes to the file before any unit and
// in a plan nowhere. Without the input, the file's last line gives it.
func TestRunTokenTakesTheInputOfTheRun(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[TokenFile] = []byte(tokenText)
	for _, c := range []struct {
		lookup func(string) (string, bool)
		want   string
	}{
		{lookupOf(map[string]string{TokenVar: "dummy-b"}), "dummy-b"},
		{lookupOf(map[string]string{TokenVar: ""}), ""},
		{lookupOf(map[string]string{}), "dummy-a"},
		{nil, "dummy-a"},
	} {
		if got, err := RunToken(m, c.lookup); err != nil || got != c.want {
			t.Errorf("want %q: got %q, %v", c.want, got, err)
		}
	}
	if got, err := RunToken(effecttest.NewMachine(), nil); err != nil || got != "" {
		t.Errorf("without the file: got %q, %v", got, err)
	}
}
