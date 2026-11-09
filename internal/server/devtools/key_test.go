package devtools

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestRepositoriesRefuseAKeyThatIsNotThePinnedOne(t *testing.T) {
	a, b := newKey(t, false), newKey(t, true)
	f := newFixture(t)
	notPinned := "not " + f.keys["one"] + " alone"
	tests := []struct{ name, body, want string }{
		{"another key", armoredKey(public(t, a)), notPinned},
		{"two certificates", armoredKey(append(public(t, a), public(t, b)...)), notPinned},
		{"no armor", "<html>", "not armored public key blocks alone"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f.m.Results[curl+"https://example.org/one.key"] = effecttest.Result{Out: []byte(tt.body)}
			if _, err := f.plan(t); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("got %v, want an error with %q", err, tt.want)
			}
		})
	}
}

func TestRepositoriesFailWithoutTheKey(t *testing.T) {
	f := newFixture(t)
	delete(f.m.Results, curl+"https://example.org/one.key")
	if _, err := f.plan(t); err == nil || !strings.Contains(err.Error(), "fetch https://example.org/one.key") {
		t.Errorf("got %v", err)
	}
	f = newFixture(t)
	delete(f.m.Files, "/usr/lib/os-release")
	if _, err := f.plan(t); err == nil {
		t.Error("a missing os-release planned")
	}
}
