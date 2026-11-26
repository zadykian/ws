package keyboard_test

import (
	"bytes"
	"encoding/xml"
	"regexp"
	"testing"
	"unicode/utf16"
)

// keylayout is what a keylayout file's keys reach: its layouts' key map sets, its keys' actions,
// and the states that an action's when sets.
type keylayout struct {
	MaxOut  int `xml:"maxout,attr"`
	Layouts []struct {
		MapSet string `xml:"mapSet,attr"`
	} `xml:"layouts>layout"`
	Sets []struct {
		ID   string `xml:"id,attr"`
		Maps []struct {
			Base string `xml:"baseMapSet,attr"`
			Keys []struct {
				Code   string `xml:"code,attr"`
				Output string `xml:"output,attr"`
				Action string `xml:"action,attr"`
			} `xml:"key"`
		} `xml:"keyMap"`
	} `xml:"keyMapSet"`
	Actions []struct {
		ID    string `xml:"id,attr"`
		Whens []when `xml:"when"`
	} `xml:"actions>action"`
	Terminators []when `xml:"terminators>when"`
}

// when is a step of an action or of the terminators: in State, it types Output or goes to Next.
type when struct {
	State  string `xml:"state,attr"`
	Next   string `xml:"next,attr"`
	Output string `xml:"output,attr"`
}

// control is a reference to a control code that XML 1.1 allows and XML 1.0 does not.
var control = regexp.MustCompile(`(?i)&#x00(0[0-8bcef]|1[0-9a-f]);`)

// functionKeys are the key codes of F1 to F20, in order. Each types the function key's code 0x10,
// which readLayout reads as U+E010.
var functionKeys = []string{
	"122", "120", "99", "118", "96", "97", "98", "100", "101", "109",
	"103", "111", "105", "107", "113", "106", "64", "79", "80", "90",
}

// readLayout parses the role's keylayout NAME. Go reads XML 1.0 alone, so the declaration says
// 1.0, and each control code becomes a private use character, one UTF-16 unit as the code is.
func readLayout(t *testing.T, name string) keylayout {
	text := roleFile(t, source+"Contents/Resources/"+name+".keylayout")
	text = bytes.Replace(text, []byte(`<?xml version="1.1"`), []byte(`<?xml version="1.0"`), 1)
	text = control.ReplaceAll(text, []byte("&#xE0$1;"))
	var k keylayout
	if err := xml.Unmarshal(text, &k); err != nil {
		t.Fatalf("%s.keylayout: %v", name, err)
	}
	return k
}

// TestTheLayoutsHoldWhatTheirKeysReach keeps each keylayout to what its keys reach. Each key map
// set is a layout's or another set's base, and each action a key's. Each state, the terminators'
// too, is one that an action goes to. maxout, the most UTF-16 units a key types, is the longest output.
func TestTheLayoutsHoldWhatTheirKeysReach(t *testing.T) {
	for _, name := range []string{"ENG", "RUS"} {
		k := readLayout(t, name)
		sets, actions, states := map[string]bool{}, map[string]bool{}, map[string]bool{"none": true}
		longest := 0
		typed := func(s string) { longest = max(longest, len(utf16.Encode([]rune(s)))) }
		for _, l := range k.Layouts {
			sets[l.MapSet] = true
		}
		for _, s := range k.Sets {
			for _, m := range s.Maps {
				sets[m.Base] = sets[m.Base] || m.Base != ""
				for _, key := range m.Keys {
					actions[key.Action] = actions[key.Action] || key.Action != ""
					typed(key.Output)
				}
			}
		}
		whens := k.Terminators
		for _, a := range k.Actions {
			if !actions[a.ID] {
				t.Errorf("%s: action %s is no key's", name, a.ID)
			}
			whens = append(whens, a.Whens...)
		}
		for _, w := range whens {
			states[w.Next] = states[w.Next] || w.Next != ""
		}
		for _, w := range whens {
			if !states[w.State] {
				t.Errorf("%s: no action goes to state %s, which outputs %q", name, w.State, w.Output)
			}
			typed(w.Output)
		}
		for _, s := range k.Sets {
			if !sets[s.ID] {
				t.Errorf("%s: key map set %s is no layout's", name, s.ID)
			}
		}
		if k.MaxOut != longest {
			t.Errorf("%s: maxout is %d, and the longest output %d", name, k.MaxOut, longest)
		}
	}
}

// TestEachMapTypesTheFunctionKeys keeps F1 to F20 in each key map of each layout, so that no
// modifier and no keyboard with F17 to F19 leaves a function key typing nothing.
func TestEachMapTypesTheFunctionKeys(t *testing.T) {
	for _, name := range []string{"ENG", "RUS"} {
		for _, s := range readLayout(t, name).Sets {
			for i, m := range s.Maps {
				typed := map[string]string{}
				for _, key := range m.Keys {
					typed[key.Code] = key.Output
				}
				for n, code := range functionKeys {
					if typed[code] != "\uE010" {
						t.Errorf("%s: map %d types %q on F%d, code %s", name, i, typed[code], n+1, code)
					}
				}
			}
		}
	}
}

// TestNoKeyTypesNothing keeps each key of each map to an output or an action. An empty output
// would leave the key dead under that map's modifiers, where the other maps type a character.
func TestNoKeyTypesNothing(t *testing.T) {
	for _, name := range []string{"ENG", "RUS"} {
		for _, s := range readLayout(t, name).Sets {
			for i, m := range s.Maps {
				for _, key := range m.Keys {
					if key.Output == "" && key.Action == "" {
						t.Errorf("%s: map %d types nothing on code %s", name, i, key.Code)
					}
				}
			}
		}
	}
}
