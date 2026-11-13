package wizard

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// callsWith returns the calls of m that hold word.
func callsWith(m *effecttest.Machine, word string) []string {
	var got []string
	for _, c := range m.Calls {
		if strings.Contains(c, word) {
			got = append(got, c)
		}
	}
	return got
}

func TestGhLoginTakesTheTerminal(t *testing.T) {
	g, m := newGh(found)
	g.Tty, g.F = true, m
	m.Results["/bin/gh auth login -p https -w -h github.com -s "+ghScopes] = effecttest.Result{}
	if err := g.login(context.Background(), ghLogin{[]string{"login", "-p", "https", "-w"}}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"foreground /bin/gh auth login -p https -w -h github.com -s " + ghScopes}; !slices.Equal(m.Calls[len(m.Calls)-1:], want) {
		t.Errorf("calls = %q, want %q last", m.Calls, want)
	}
}
