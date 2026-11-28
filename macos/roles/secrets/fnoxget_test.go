package secrets_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/secrets"
)

// getLine is fnox get of ws-a's WS_IKEWARD_PASSWORD in the clone, and value what it prints.
const (
	getLine = "env -u FNOX_AGE_KEY -u FNOX_PROFILE FNOX_AGE_KEY_FILE=/h/.config/fnox/age.txt " +
		`sh -c cd "$1" && shift && exec "$@" sh ` + cloneDir + " fnox -P ws-a get WS_IKEWARD_PASSWORD"
	value = "the-value-of-the-password"
)

// getting is a clone of o/n whose fnox get of ws-a's password gives r.
func getting(t *testing.T, r effecttest.Result) (*effecttest.Machine, *secrets.Clone) {
	m := cloneable()
	m.Results[getLine] = r
	c := open(t, m)
	c.FnoxEnv = []string{"FNOX_AGE_KEY_FILE=/h/.config/fnox/age.txt"}
	return m, c
}

func TestFnoxGetReadsTheValueOnTheTerminal(t *testing.T) {
	m, c := getting(t, effecttest.Result{Out: []byte(value + "\n")})
	got, err := c.FnoxGet(context.Background(), "ws-a", "WS_IKEWARD_PASSWORD")
	if err != nil || string(got) != value {
		t.Fatalf("FnoxGet = %q, %v; want the value without fnox's newline", got, err)
	}
	if last := m.Calls[len(m.Calls)-1]; last != "terminal "+getLine {
		t.Errorf("FnoxGet ran %q, want fnox get on the terminal", last)
	}
	if len(m.Inputs) != 0 || strings.Contains(strings.Join(m.Calls, "\n"), value) {
		t.Errorf("the value reached a command: inputs %q, calls %q", m.Inputs, m.Calls)
	}
}

// fnox.toml declares each input empty, so a server without its own value reads "".
func TestFnoxGetRefusesAnEmptyValue(t *testing.T) {
	_, c := getting(t, effecttest.Result{Out: []byte("\n")})
	got, err := c.FnoxGet(context.Background(), "ws-a", "WS_IKEWARD_PASSWORD")
	if err == nil || got != nil || !strings.Contains(err.Error(), "fnox -P ws-a set WS_IKEWARD_PASSWORD --provider host") {
		t.Errorf("an empty value gives %q, %v; want an error naming the set of ws-a's own", got, err)
	}
}

func TestFnoxGetKeepsTheValueOutOfItsError(t *testing.T) {
	_, c := getting(t, effecttest.Result{Out: []byte(value + "\n"), Err: errors.New("exit status 1")})
	got, err := c.FnoxGet(context.Background(), "ws-a", "WS_IKEWARD_PASSWORD")
	if err == nil || got != nil || strings.Contains(err.Error(), value) {
		t.Errorf("a failed fnox get gives %q, %v; want an error without the value", got, err)
	}
}

// A profile or a name that is not of its form could reach fnox as a flag.
func TestFnoxGetRefusesAProfileOrANameOfAnotherForm(t *testing.T) {
	for _, in := range [][2]string{{"-x", "WS_A"}, {"nano", "WS_A"}, {"ws-a", "--all"}, {"ws-a", ""}} {
		m, c := getting(t, effecttest.Result{})
		calls := len(m.Calls)
		if _, err := c.FnoxGet(context.Background(), in[0], in[1]); err == nil || len(m.Calls) != calls {
			t.Errorf("FnoxGet(%q, %q) gives %v and ran %q; want an error before fnox", in[0], in[1], err, m.Calls[calls:])
		}
	}
}
