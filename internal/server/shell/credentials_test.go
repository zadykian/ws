package shell

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/server/values"
)

func TestCredentialFile(t *testing.T) {
	for url, want := range map[string]string{
		"https://a.example/x": "a.example_x", "https://b.example": "b.example",
		"https://h:8443/a//b?c": "h_8443_a_b_c", "https://x.y/é": "x.y_",
	} {
		if got := credentialFile(url); got != want {
			t.Errorf("%s: %s, want %s", url, got, want)
		}
	}
}

func TestSectionsKeepTheOmittedAndFollowTheEntries(t *testing.T) {
	keys := []string{
		"user.name", "credential.https://kept.example.helper", "http.https://kept.example.proactiveauth",
		"credential.https://old.example.helper", "http.https://old.example.proactiveauth",
		"credential.https://new.example.helper",
	}
	creds := []values.GitCredential{
		{URL: "https://old.example", State: "absent"},
		{URL: "https://new.example", ProactiveAuth: true},
		{URL: "https://another.example"},
	}
	want := []section{
		{"https://another.example", "another.example", false},
		{"https://kept.example", "kept.example", true},
		{"https://new.example", "new.example", true},
	}
	if got := sections(keys, creds); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := keptURLs(keys, creds); !slices.Equal(got, []string{"https://kept.example"}) {
		t.Errorf("kept = %v", got)
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
	ok := values.GitCredential{URL: "https://a.example/x", Token: "t"}
	kept := []string{"credential.https://a.example_x.helper"}
	cases := map[string]struct {
		creds []values.GitCredential
		keys  []string
		bad   bool
	}{
		"good":                  {[]values.GitCredential{ok, {URL: "https://b.example", Token: "t", Username: "u", State: "present"}}, nil, false},
		"absent needs no token": {[]values.GitCredential{{URL: "https://b.example", State: "absent"}}, nil, false},
		"http":                  {[]values.GitCredential{{URL: "http://a.example", Token: "t"}}, nil, true},
		"quote in url":          {[]values.GitCredential{{URL: `https://a"b`, Token: "t"}}, nil, true},
		"space in url":          {[]values.GitCredential{{URL: "https://a b", Token: "t"}}, nil, true},
		"state":                 {[]values.GitCredential{{URL: "https://a.b", Token: "t", State: "gone"}}, nil, true},
		"no token":              {[]values.GitCredential{{URL: "https://a.b"}}, nil, true},
		"token space":           {[]values.GitCredential{{URL: "https://a.b", Token: "a b"}}, nil, true},
		"username":              {[]values.GitCredential{{URL: "https://a.b", Token: "t", Username: "a\tb"}}, nil, true},
		"same url twice":        {[]values.GitCredential{ok, ok}, nil, true},
		"same file":             {[]values.GitCredential{{URL: "https://a.b/c", Token: "t"}, {URL: "https://a.b?c", Token: "t"}}, nil, true},
		"kept clash":            {[]values.GitCredential{{URL: "https://a.example/x", Token: "t", State: "absent"}, {URL: "https://z", Token: "t"}}, kept, true},
	}
	for name, c := range cases {
		err := checkCredentials(c.creds, c.keys)
		if (err != nil) != c.bad {
			t.Errorf("%s: %v, want an error: %v", name, err, c.bad)
		}
		if err != nil && strings.Contains(err.Error(), "hunter") {
			t.Errorf("%s: the message shows a token", name)
		}
	}
}
