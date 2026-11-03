package shell

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/values"
)

func tokenFixture(creds ...values.GitCredential) (*effecttest.Machine, tokens) {
	m := effecttest.NewMachine()
	m.Results["git config --file /home/ws/.config/git/tokens.gitconfig.ws-check --name-only --list"] = effecttest.Result{}
	m.Results["git config --file - --name-only --list"] = effecttest.Result{
		Out: []byte("credential.https://kept.example.helper\n")}
	a := account{User: host.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000}
	return m, tokens{M: m, A: a, Creds: creds}
}

func applyTokens(t *testing.T, tk tokens) []string {
	changes, err := tk.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var shown []string
	for _, c := range changes {
		shown = append(shown, c.String())
		if err := c.Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	return shown
}

func TestTokensAreWrittenOnceAndNeverShown(t *testing.T) {
	m, tk := tokenFixture(
		values.GitCredential{URL: "https://a.example", Token: "hunter2", ProactiveAuth: true},
		values.GitCredential{URL: "https://b.example", Token: "hunter3", Username: "bob"},
		values.GitCredential{URL: "https://gone.example", State: "absent"},
	)
	m.Files["/home/ws/.config/git/tokens/gone.example"] = []byte("old")
	m.Files["/home/ws/.config/git/tokens.gitconfig"] = []byte("[credential \"https://kept.example\"]\n")
	for _, s := range applyTokens(t, tk) {
		if strings.Contains(s, "hunter") {
			t.Errorf("a change shows a token: %q", s)
		}
	}
	if got := string(m.Files["/home/ws/.config/git/tokens/b.example"]); got != "username=bob\npassword=hunter3\n" {
		t.Errorf("b's token file = %q", got)
	}
	if got := string(m.Files["/home/ws/.config/git/tokens/a.example"]); got != "username=token\npassword=hunter2\n" {
		t.Errorf("a's token file = %q", got)
	}
	if _, ok := m.Files["/home/ws/.config/git/tokens/gone.example"]; ok {
		t.Error("the token of an absent entry stays")
	}
	cfg := string(m.Files["/home/ws/.config/git/tokens.gitconfig"])
	if !strings.Contains(cfg, `[credential "https://kept.example"]`) || strings.Contains(cfg, "gone.example") || !strings.Contains(cfg, `[http "https://a.example"]`) {
		t.Errorf("tokens.gitconfig = %q", cfg)
	}
	if again := applyTokens(t, tk); again != nil {
		t.Errorf("second plan = %q, want none", again)
	}
}

func TestTokensLeaveEverythingWithoutEntries(t *testing.T) {
	m, tk := tokenFixture()
	if got := applyTokens(t, tk); got != nil || len(m.Calls) != 0 {
		t.Errorf("plan = %q, calls %q", got, m.Calls)
	}
}

func TestTokensRefuseBeforeWriting(t *testing.T) {
	m, tk := tokenFixture(values.GitCredential{URL: "https://a.example", Token: "has space"})
	if _, err := tk.Plan(context.Background()); err == nil || strings.Contains(err.Error(), "has space") {
		t.Errorf("got %v", err)
	}
	if len(m.Files) != 0 {
		t.Error("a file was written")
	}
}
