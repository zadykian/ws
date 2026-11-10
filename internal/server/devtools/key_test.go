package devtools

import (
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/zadykian/ws/effect/effecttest"
)

// pin makes the fixture's repository one pin the keys alone.
func pin(f fixture, keys ...*openpgp.Entity) {
	var prints []string
	for _, e := range keys {
		prints = append(prints, upper(e.PrimaryKey.Fingerprint))
	}
	f.repos("resolute")[0].Fingerprints = prints
}

func TestRepositoriesTakeAKeyWhoseCertificatesAreAllPinned(t *testing.T) {
	a, b := newKey(t, false), newKey(t, true)
	f := newFixture(t)
	pin(f, a, b)
	tests := []struct{ name, body string }{
		{"both, in one block", armoredKey(append(public(t, a), public(t, b)...))},
		{"both, in two blocks", armoredKey(public(t, a)) + armoredKey(public(t, b))},
		{"one, as the vendor dropped the other", armoredKey(public(t, b))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f.m.Results[curl+"https://example.org/one.key"] = effecttest.Result{Out: []byte(tt.body)}
			if _, err := f.plan(t); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestRepositoriesRefuseAKeyThatIsNotPinned(t *testing.T) {
	a, b := newKey(t, false), newKey(t, true)
	f := newFixture(t)
	pin(f, a)
	notPinned := upper(b.PrimaryKey.Fingerprint) + ", which is not pinned"
	tests := []struct{ name, body, want string }{
		{"another key", armoredKey(public(t, b)), notPinned},
		{"another key beside the pinned one", armoredKey(append(public(t, a), public(t, b)...)), notPinned},
		{"another key in a second block", armoredKey(public(t, a)) + armoredKey(public(t, b)), notPinned},
		{"no key", armoredKey(nil), "served no key"},
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
