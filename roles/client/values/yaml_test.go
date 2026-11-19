package values

import (
	"reflect"
	"testing"
)

func TestEntries(t *testing.T) {
	tests := []struct {
		name, text string
		want       []entry
		errs       []Error
	}{
		{name: "empty"},
		{name: "comments", text: "# a\n\n  # b\n"},
		{"pairs", "WS_USER: ada\nWS_SLOT: 3\nWS_LAN: \"10.88.0.0/24\"\n", []entry{
			{key: "WS_USER", value: "ada", line: 1},
			{key: "WS_SLOT", value: "3", line: 2},
			{key: "WS_LAN", value: "10.88.0.0/24", line: 3},
		}, nil},
		{"blank in a bare value", "# c\nWS_CABLE_SERVICE: USB LAN\n", []entry{
			{key: "WS_CABLE_SERVICE", value: "USB LAN", line: 2},
		}, nil},
		{"empty values", "WS_USER:\nWS_TARGET: \"\"\nWS_LAN: ~\nWS_REMOTE: null\n", []entry{
			{key: "WS_USER", line: 1}, {key: "WS_TARGET", line: 2},
			{key: "WS_LAN", line: 3}, {key: "WS_REMOTE", line: 4},
		}, nil},
		{"quoted null is text", "WS_USER: \"null\"\n", []entry{{key: "WS_USER", value: "null", line: 1}}, nil},
		{"key twice", "WS_USER: a\nWS_SLOT: 1\nWS_USER: b\n", []entry{
			{key: "WS_USER", value: "a", line: 1}, {key: "WS_SLOT", value: "1", line: 2},
			{key: "WS_USER", err: "WS_USER is set already, on line 1", line: 3},
		}, nil},
		{"a list", "WS_USER: [a]\n", []entry{
			{key: "WS_USER", err: "WS_USER takes one value, not a list or a table", line: 1},
		}, nil},
		{"a table", "WS_USER: {a: b}\n", []entry{
			{key: "WS_USER", err: "WS_USER takes one value, not a list or a table", line: 1},
		}, nil},
		{"a list file", "- WS_USER\n", nil, []Error{{1, oneLine}}},
		{"a scalar file", "WS_USER\n", nil, []Error{{1, oneLine}}},
		{"syntax", "WS_USER: a\nWS_SLOT: \"1\n", nil, []Error{{2, "not YAML: found unexpected end of stream"}}},
		{"a flow list open", "WS_SLOT: 1\nWS_USER: [a\n", nil, []Error{{2, "not YAML: did not find expected ',' or ']'"}}},
		{"two values", "WS_USER: a: b\n", nil, []Error{{1, "not YAML: mapping values are not allowed in this context"}}},
	}
	for _, tt := range tests {
		got, errs := entries(tt.text)
		if !reflect.DeepEqual(got, tt.want) || !reflect.DeepEqual(errs, tt.errs) {
			t.Errorf("%s: %+v, %v; want %+v, %v", tt.name, got, errs, tt.want, tt.errs)
		}
	}
}

func TestEntriesRefuseWhatTheWritersCannotEdit(t *testing.T) {
	for name, tc := range map[string]struct {
		text string
		line int
	}{
		"a key in quotes":  {"WS_USER: a\n\"WS_SLOT\": 1\n", 2},
		"a space":          {"WS_USER: a\nWS_SLOT : 1\n", 2},
		"a flow map":       {"WS_USER: a\n{WS_SLOT: 1}\n", 2},
		"a question mark":  {"? WS_USER\n: a\n", 1},
		"a BOM":            {"\ufeffWS_USER: a\n", 1},
		"a wrapped value":  {"WS_USER: a\n  b\n", 2},
		"a value below":    {"WS_USER:\n  a\n", 2},
		"a folded value":   {"WS_USER: >-\n  a\n  b\n", 1},
		"a literal value":  {"WS_USER: |\n  a\n", 1},
		"a quoted wrap":    {"WS_USER: 'a\n  b'\n", 1},
		"a document start": {"---\nWS_USER: a\n", 1},
		"a second doc":     {"WS_USER: a\n---\nWS_SLOT: 1\n", 2},
		"a document end":   {"WS_USER: a\n...\n", 2},
		"an indented key":  {"WS_USER: a\n  WS_SLOT: 1\n", 2},
	} {
		got, errs := entries(tc.text)
		if len(errs) != 1 || errs[0].Line != tc.line || got != nil {
			t.Errorf("%s: %+v, %v; want one refusal on line %d", name, got, errs, tc.line)
		}
	}
}
