package values

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestReplace(t *testing.T) {
	for name, tc := range map[string]struct{ text, want string }{
		"missing":  {"A: 1\n", "A: 1\nK: v\n"},
		"no break": {"A: 1", "A: 1\nK: v\n"},
		"set":      {"A: 1\nK: old\nB: 2\n", "A: 1\nK: v\nB: 2\n"},
		"twice":    {"K: a\nA: 1\nK: b\n", "K: v\nA: 1\n"},
		"comment":  {"# K: x\nK: a\n", "# K: x\nK: v\n"},
		"empty":    {"", "K: v\n"},
	} {
		if got := replace(tc.text, "K", "v"); got != tc.want {
			t.Errorf("%s: replace = %q; want %q", name, got, tc.want)
		}
	}
}

func TestReplaceQuotesWhereYAMLNeedsIt(t *testing.T) {
	for value, want := range map[string]string{"a b": "K: a b\n", "a: b": "K: \"a: b\"\n", "": "K: \"\"\n"} {
		if got := replace("K: old\n", "K", value); got != want {
			t.Errorf("replace %q = %q; want %q", value, got, want)
		}
	}
}

func TestConfSetAllPassesAPairTogether(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/values.yml"] = []byte(Text("WS_TARGET", "lan"))
	m.Results["mkdir -p /h"] = effecttest.Result{}
	f := File{Path: "/h/values.yml", Stage: "Configure", M: m, Out: &effecttest.Out{}}
	if err := f.Set(context.Background(), "WS_LAN", "10.88.0.0/24"); err == nil {
		t.Error("set of half a pair = nil; want a refusal")
	}
	pair := []Setting{{"WS_LAN", "10.88.0.0/24"}, {"WS_LAN_ADDRESS", "10.88.0.10"}}
	if err := f.SetAll(context.Background(), pair...); err != nil {
		t.Fatal(err)
	}
	want := Text("WS_TARGET", "lan", "WS_LAN", "10.88.0.0/24", "WS_LAN_ADDRESS", "10.88.0.10")
	if got := string(m.Files["/h/values.yml"]); got != want {
		t.Errorf("values.yml = %q; want %q", got, want)
	}
}

func TestConfSet(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/values.yml"] = []byte(Text("WS_TARGET", "lan", "WS_PUBLIC_KEY", "x"))
	m.Results["mkdir -p /h"] = effecttest.Result{}
	f := File{Path: "/h/values.yml", Stage: "Tunnel", M: m, Out: &effecttest.Out{}}
	if err := f.Set(context.Background(), "WS_PUBLIC_KEY", "bad"); err == nil {
		t.Error("set of a bad key = nil; want a refusal")
	}
	key := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	if err := f.Set(context.Background(), "WS_PUBLIC_KEY", key); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/h/values.yml"]); got != Text("WS_TARGET", "lan", "WS_PUBLIC_KEY", key) {
		t.Errorf("values.yml = %q", got)
	}
}

func TestConfSetRefusesAWrappedValue(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/values.yml"] = []byte("WS_TARGET: lan\nWS_TUNNEL_NETWORKS: 10.1.0.0/16\n  10.2.0.0/16\n")
	m.Results["mkdir -p /h"] = effecttest.Result{}
	var out effecttest.Out
	f := File{Path: "/h/values.yml", Stage: "Configure", M: m, Out: &out}
	if err := f.Set(context.Background(), "WS_TUNNEL_NETWORKS", "10.3.0.0/16"); !errors.Is(err, ErrRefused) {
		t.Errorf("set of a wrapped value: %v; want a refusal", err)
	}
	if len(out.Lines) != 1 || !strings.Contains(out.Lines[0], "values.yml, line 3: write one KEY: value a line") {
		t.Errorf("out = %v; want the refusal of line 3", out.Lines)
	}
}
