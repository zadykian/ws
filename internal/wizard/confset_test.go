package wizard

import (
	"context"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
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
