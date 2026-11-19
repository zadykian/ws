package git

import (
	"os"
	"strings"
	"testing"
)

func TestCredentialKey(t *testing.T) {
	for url, want := range map[string]string{
		"https://a.example/x": "a.example_x", "https://b.example": "b.example",
		"https://h:8443/a//b?c": "h_8443_a_b_c", "https://x.y/é": "x.y_",
	} {
		if got := credentialKey(url); got != want {
			t.Errorf("%s: %s, want %s", url, got, want)
		}
	}
}

func TestTokensConfigIsTheRolesRendering(t *testing.T) {
	got, err := tokensConfig([]section{
		{"https://a.example/x", "a.example_x", false}, {"https://b.example", "b.example", true},
	})
	want, rerr := os.ReadFile("testdata/tokens.gitconfig")
	if err != nil || rerr != nil || string(got) != string(want) {
		t.Errorf("got %q (%v), want the role's rendering (%v)", got, err, rerr)
	}
}

func TestCheckCredentials(t *testing.T) {
	ok := Credential{URL: "https://a.example/x", Token: "t"}
	cases := map[string]struct {
		creds []Credential
		bad   bool
	}{
		"good":           {[]Credential{ok, {URL: "https://b.example", Token: "t", Username: "u"}}, false},
		"http":           {[]Credential{{URL: "http://a.example", Token: "hunter2"}}, true},
		"quote in url":   {[]Credential{{URL: `https://a"b`, Token: "hunter2"}}, true},
		"space in url":   {[]Credential{{URL: "https://a b", Token: "hunter2"}}, true},
		"no token":       {[]Credential{{URL: "https://a.b"}}, true},
		"token space":    {[]Credential{{URL: "https://a.b", Token: "hunter 2"}}, true},
		"username":       {[]Credential{{URL: "https://a.b", Token: "hunter2", Username: "a\tb"}}, true},
		"same url twice": {[]Credential{ok, ok}, true},
		"same key":       {[]Credential{{URL: "https://a.b/c", Token: "t"}, {URL: "https://a.b?c", Token: "t"}}, true},
	}
	for name, c := range cases {
		err := checkCredentials(c.creds)
		if (err != nil) != c.bad {
			t.Errorf("%s: %v, want an error: %v", name, err, c.bad)
		}
		if err != nil && strings.Contains(err.Error(), "hunter") {
			t.Errorf("%s: the message shows a token", name)
		}
	}
}
