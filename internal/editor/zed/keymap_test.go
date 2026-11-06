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

func TestASectionRendersItsContextAndUnbinds(t *testing.T) {
	chord, err := editor.ParseChord("ctrl+a")
	if err != nil {
		t.Fatal(err)
	}
	keymap := editor.Keymap{editor.Text: {{Action: "a.plain", Keys: []editor.Chord{chord}}}}
	targets := map[string]editor.Target{"a.plain": {To: "x::Plain"}}
	l := layout{Header: "Mark\n", Sections: []section{
		{
			Scope: editor.Text, Context: "Editor", Comment: "Second.\n",
			Unbind: []unbind{{Key: "ctrl-k", To: "x::Cut"}},
		},
	}}
	want := `// Mark
[
  // Second.
  {
    "context": "Editor",
    "unbind": {
      "ctrl-k": "x::Cut"
    },
    "bindings": {
      "ctrl-a": "x::Plain"
    }
  }
]
`
	if got := string(render(keymap, targets, l)); got != want {
		t.Errorf("render =\n%s\nwant\n%s", got, want)
	}
}
