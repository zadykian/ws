package wizard

import (
	"context"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestReplace(t *testing.T) {
	for name, tc := range map[string]struct{ text, want string }{
		"missing":  {"A=1\n", "A=1\nK=v\n"},
		"no break": {"A=1", "A=1\nK=v\n"},
		"set":      {"A=1\nK=old\nB=2\n", "A=1\nK=v\nB=2\n"},
		"twice":    {"K=a\nA=1\nK=b\n", "K=v\nA=1\n"},
		"empty":    {"", "K=v\n"},
	} {
		if got := replace(tc.text, "K", "v"); got != tc.want {
			t.Errorf("%s: replace = %q; want %q", name, got, tc.want)
		}
	}
}

func TestReplaceQuotesABlank(t *testing.T) {
	if got, want := replace("K=old\n", "K", "a b"), "K=\"a b\"\n"; got != want {
		t.Errorf("replace = %q; want %q", got, want)
	}
}

func TestConfSetAllPassesAPairTogether(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\n")
	m.Results["mkdir -p /h"] = effecttest.Result{}
	f := confFile{"/h/mac.conf", "Configure", m, &effecttest.Out{}}
	if err := f.set(context.Background(), "WS_LAN", "10.88.0.0/24"); err == nil {
		t.Error("set of half a pair = nil; want a refusal")
	}
	pair := []setting{{"WS_LAN", "10.88.0.0/24"}, {"WS_LAN_ADDRESS", "10.88.0.10"}}
	if err := f.setAll(context.Background(), pair...); err != nil {
		t.Fatal(err)
	}
	want := "WS_TARGET=lan\nWS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.10\n"
	if got := string(m.Files["/h/mac.conf"]); got != want {
		t.Errorf("mac.conf = %q; want %q", got, want)
	}
}

func TestConfSet(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\nWS_PUBLIC_KEY=x\n")
	m.Results["mkdir -p /h"] = effecttest.Result{}
	f := confFile{"/h/mac.conf", "Tunnel", m, &effecttest.Out{}}
	if err := f.set(context.Background(), "WS_PUBLIC_KEY", "bad"); err == nil {
		t.Error("set of a bad key = nil; want a refusal")
	}
	key := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	if err := f.set(context.Background(), "WS_PUBLIC_KEY", key); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/h/mac.conf"]); got != "WS_TARGET=lan\nWS_PUBLIC_KEY="+key+"\n" {
		t.Errorf("mac.conf = %q", got)
	}
}
