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
		{"pairs", "WS_TUNNEL_PORT: ada\nWS_SLOT: 3\nWS_LAN: \"10.88.0.0/24\"\n", []entry{
			{key: "WS_TUNNEL_PORT", value: "ada", line: 1},
			{key: "WS_SLOT", value: "3", line: 2},
			{key: "WS_LAN", value: "10.88.0.0/24", line: 3},
		}, nil},
		{"blank in a bare value", "# c\nWS_CABLE_SERVICE: USB LAN\n", []entry{
			{key: "WS_CABLE_SERVICE", value: "USB LAN", line: 2},
		}, nil},
		{"empty values", "WS_TUNNEL_PORT:\nWS_TARGET: \"\"\nWS_LAN: ~\nWS_REMOTE: null\n", []entry{
			{key: "WS_TUNNEL_PORT", line: 1}, {key: "WS_TARGET", line: 2},
			{key: "WS_LAN", line: 3}, {key: "WS_REMOTE", line: 4},
		}, nil},
		{"quoted null is text", "WS_TUNNEL_PORT: \"null\"\n", []entry{{key: "WS_TUNNEL_PORT", value: "null", line: 1}}, nil},
		{"key twice", "WS_TUNNEL_PORT: a\nWS_SLOT: 1\nWS_TUNNEL_PORT: b\n", []entry{
			{key: "WS_TUNNEL_PORT", value: "a", line: 1}, {key: "WS_SLOT", value: "1", line: 2},
			{key: "WS_TUNNEL_PORT", err: "WS_TUNNEL_PORT is set already, on line 1", line: 3},
		}, nil},
		{"a list", "WS_TUNNEL_PORT: [a]\n", []entry{
			{key: "WS_TUNNEL_PORT", err: "WS_TUNNEL_PORT takes one value, not a list or a table", line: 1},
		}, nil},
		{"a table", "WS_TUNNEL_PORT: {a: b}\n", []entry{
			{key: "WS_TUNNEL_PORT", err: "WS_TUNNEL_PORT takes one value, not a list or a table", line: 1},
		}, nil},
		{"a list file", "- WS_TUNNEL_PORT\n", nil, []Error{{1, oneLine}}},
		{"a scalar file", "WS_TUNNEL_PORT\n", nil, []Error{{1, oneLine}}},
		{"syntax", "WS_TUNNEL_PORT: a\nWS_SLOT: \"1\n", nil, []Error{{2, "not YAML: found unexpected end of stream"}}},
		{"a flow list open", "WS_SLOT: 1\nWS_TUNNEL_PORT: [a\n", nil, []Error{{2, "not YAML: did not find expected ',' or ']'"}}},
		{"two values", "WS_TUNNEL_PORT: a: b\n", nil, []Error{{1, "not YAML: mapping values are not allowed in this context"}}},
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
		"a key in quotes":  {"WS_TUNNEL_PORT: a\n\"WS_SLOT\": 1\n", 2},
		"a space":          {"WS_TUNNEL_PORT: a\nWS_SLOT : 1\n", 2},
		"a flow map":       {"WS_TUNNEL_PORT: a\n{WS_SLOT: 1}\n", 2},
		"a question mark":  {"? WS_TUNNEL_PORT\n: a\n", 1},
		"a BOM":            {"\ufeffWS_TUNNEL_PORT: a\n", 1},
		"a wrapped value":  {"WS_TUNNEL_PORT: a\n  b\n", 2},
		"a value below":    {"WS_TUNNEL_PORT:\n  a\n", 2},
		"a folded value":   {"WS_TUNNEL_PORT: >-\n  a\n  b\n", 1},
		"a literal value":  {"WS_TUNNEL_PORT: |\n  a\n", 1},
		"a quoted wrap":    {"WS_TUNNEL_PORT: 'a\n  b'\n", 1},
		"a document start": {"---\nWS_TUNNEL_PORT: a\n", 1},
		"a second doc":     {"WS_TUNNEL_PORT: a\n---\nWS_SLOT: 1\n", 2},
		"a document end":   {"WS_TUNNEL_PORT: a\n...\n", 2},
		"an indented key":  {"WS_TUNNEL_PORT: a\n  WS_SLOT: 1\n", 2},
	} {
		got, errs := entries(tc.text)
		if len(errs) != 1 || errs[0].Line != tc.line || got != nil {
			t.Errorf("%s: %+v, %v; want one refusal on line %d", name, got, errs, tc.line)
		}
	}
}
