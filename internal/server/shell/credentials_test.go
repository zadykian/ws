package shell

import (
	"os"
	"slices"
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
