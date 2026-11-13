package git

import (
	"slices"
	"testing"
)

func TestDropped(t *testing.T) {
	cases := []struct {
		name      string
		old, want []string
		got       []string
	}{
		{"none", []string{"user.name"}, []string{"user.name", "x.y"}, nil},
		{"plain", []string{"user.name", "core.editor"}, []string{"user.name"}, []string{"core.editor"}},
		{"subsections", []string{"http.https://a.b/c.extraheader", "http.https://d.e.extraheader", "alias.l"},
			nil, []string{"alias.l", "http.*.extraheader"}},
	}
	for _, c := range cases {
		if got := dropped(c.old, c.want); !slices.Equal(got, c.got) {
			t.Errorf("%s: %q, want %q", c.name, got, c.got)
		}
	}
}

func TestLocalSigning(t *testing.T) {
	got := localSigning([]string{"user.name", "gpg.format", "commit.gpgsign", "user.signingkey", "gpg.format", "gpgx.y"})
	if want := []string{"commit.gpgsign", "gpg.format", "user.signingkey"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSigningKeyOf(t *testing.T) {
	list := []string{"user.name=a", "user.signingkey=~/k", "user.signingkey=~/.ssh/s.pub"}
	if k, ok := signingKeyOf(list, "/home/ws"); !ok || k != "/home/ws/.ssh/s.pub" {
		t.Errorf("got %q, %v", k, ok)
	}
	if _, ok := signingKeyOf([]string{"user.name=a"}, "/h"); ok {
		t.Error("found a key in a list without one")
	}
}
