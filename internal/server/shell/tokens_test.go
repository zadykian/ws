package shell

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func tokenFixture(creds ...credential) (*effecttest.Machine, tokens) {
	m := effecttest.NewMachine()
	m.Results["git config --file /home/ws/.config/git/tokens.gitconfig.ws-check --name-only --list"] = effecttest.Result{}
	m.Results["git config --file - --name-only --list"] = effecttest.Result{
		Out: []byte("credential.https://kept.example.helper\n")}
	a := host.Account{User: host.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000}
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
		credential{URL: "https://a.example", Token: "hunter2", ProactiveAuth: true},
		credential{URL: "https://b.example", Token: "hunter3", Username: "bob"},
	)
	m.Files["/home/ws/.config/git/tokens/kept.example"] = []byte("old")
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
	if got := string(m.Files["/home/ws/.config/git/tokens/kept.example"]); got != "old" {
		t.Errorf("the token of an omitted URL = %q, want it kept", got)
	}
	cfg := string(m.Files["/home/ws/.config/git/tokens.gitconfig"])
	if !strings.Contains(cfg, `[credential "https://kept.example"]`) || !strings.Contains(cfg, `[http "https://a.example"]`) {
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
	m, tk := tokenFixture(credential{URL: "https://a.example", Token: "has space"})
	if _, err := tk.Plan(context.Background()); err == nil || strings.Contains(err.Error(), "has space") {
		t.Errorf("got %v", err)
	}
	if len(m.Files) != 0 {
		t.Error("a file was written")
	}
}

func TestTokensOfADroppedURLGoWithTheWholeTruth(t *testing.T) {
	m, tk := tokenFixture(credential{URL: "https://a.example", Token: "hunter2"})
	tk.Whole = true
	m.Files["/home/ws/.config/git/tokens/kept.example"] = []byte("old")
	m.Files["/home/ws/.config/git/tokens.gitconfig"] = []byte("[credential \"https://kept.example\"]\n")
	for _, s := range applyTokens(t, tk) {
		if strings.Contains(s, "hunter") {
			t.Errorf("a change shows a token: %q", s)
		}
	}
	if _, ok := m.Files["/home/ws/.config/git/tokens/kept.example"]; ok {
		t.Error("the token of a dropped URL stays")
	}
	if cfg := string(m.Files["/home/ws/.config/git/tokens.gitconfig"]); strings.Contains(cfg, "kept.example") ||
		!strings.Contains(cfg, `[credential "https://a.example"]`) {
		t.Errorf("tokens.gitconfig = %q", cfg)
	}
	if again := applyTokens(t, tk); again != nil {
		t.Errorf("second plan = %q, want none", again)
	}
}

func TestTokensWithAnEmptyTableAndNoFileChangeNothing(t *testing.T) {
	_, tk := tokenFixture()
	tk.Whole = true
	if got := applyTokens(t, tk); got != nil {
		t.Errorf("plan = %q", got)
	}
}

func TestTokensWithAnEmptyTableRemoveWhatTheFileHeld(t *testing.T) {
	m, tk := tokenFixture()
	tk.Whole = true
	m.Files["/home/ws/.config/git/tokens/kept.example"] = []byte("old")
	m.Files["/home/ws/.config/git/tokens.gitconfig"] = []byte("[credential \"https://kept.example\"]\n")
	applyTokens(t, tk)
	if _, ok := m.Files["/home/ws/.config/git/tokens/kept.example"]; ok {
		t.Error("the token stays")
	}
	if cfg := string(m.Files["/home/ws/.config/git/tokens.gitconfig"]); strings.Contains(cfg, "kept.example") {
		t.Errorf("tokens.gitconfig = %q", cfg)
	}
}

func TestTokensRefuseALinkAtATokenFile(t *testing.T) {
	m, tk := tokenFixture(credential{URL: "https://a.example", Token: "hunter2"})
	m.Links["/home/ws/.config/git/tokens/a.example"] = "/etc/shadow"
	if _, err := tk.Plan(context.Background()); err == nil {
		t.Error("no error for a link at a token file")
	}
}
