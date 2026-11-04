package zed

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/internal/editor"
)

func TestABindingRendersItsArguments(t *testing.T) {
	chord := func(text string) editor.Chord {
		c, err := editor.ParseChord(text)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	list := []editor.Binding{
		{Action: "a.plain", Keys: []editor.Chord{chord("ctrl+a")}},
		{Action: "a.args", Keys: []editor.Chord{chord("ctrl+b")}},
		{Action: "a.key", Keys: []editor.Chord{chord("ctrl+c"), chord("ctrl+d")}},
		{Action: "a.reason", Keys: []editor.Chord{chord("ctrl+e")}},
	}
	targets := map[string]editor.Target{
		"a.plain":  {To: "x::Plain"},
		"a.args":   {To: "x::Args", Args: `{ "n": 1 }`},
		"a.key":    {To: "x::Send", KeyArg: true},
		"a.reason": {Why: "none"},
	}
	want := []string{
		`"ctrl-a": "x::Plain"`,
		`"ctrl-b": ["x::Args", { "n": 1 }]`,
		`"ctrl-c": ["x::Send", "ctrl-c"]`,
		`"ctrl-d": ["x::Send", "ctrl-d"]`,
	}
	if got := bindings(list, targets); !slices.Equal(got, want) {
		t.Errorf("bindings = %q; want %q", got, want)
	}
}
