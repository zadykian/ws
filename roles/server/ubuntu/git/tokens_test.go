package git

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/account"
)

// tokenFixture is a machine whose TokensFile holds text, where text is not empty, and the Tokens of
// the account ws on it.
func tokenFixture(text string) (*effecttest.Machine, Tokens) {
	m := effecttest.NewMachine()
	if text != "" {
		m = tokensMachine(text)
	}
	m.Results["git config --file /etc/ws/tokens.gitconfig.ws-check --name-only --list"] = effecttest.Result{}
	a := account.Account{User: account.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000}
	return m, Tokens{M: m, A: a}
}

func applyTokens(t *testing.T, tk Tokens) []string {
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

// goldenTokens is the file of testdata/tokens.gitconfig.
const goldenTokens = `["https://b.example"]
token = "` + dummy + `-b"
proactive_auth = true

["https://a.example/x"]
token = "` + dummy + `-a"
`

func TestTokensConfigNamesWsAsTheProgram(t *testing.T) {
	m, tk := tokenFixture(goldenTokens)
	for _, s := range applyTokens(t, tk) {
		if strings.Contains(s, dummy) {
			t.Errorf("a change shows a token: %q", s)
		}
	}
	want, err := os.ReadFile("testdata/tokens.gitconfig")
	if got := m.Files[tokensPath]; err != nil || string(got) != string(want) {
		t.Errorf("tokens.gitconfig = %q (%v), want the role's rendering", got, err)
	}
	if again := applyTokens(t, tk); again != nil {
		t.Errorf("second plan = %q, want none", again)
	}
}

func TestTokensWritesNoFilePerURL(t *testing.T) {
	m, tk := tokenFixture(twoTokens)
	applyTokens(t, tk)
	for p := range m.Files {
		if p != TokensFile && p != tokensPath {
			t.Errorf("wrote %s", p)
		}
	}
	if len(m.Dirs) != 0 {
		t.Errorf("made %v", m.Dirs)
	}
	if strings.Contains(string(m.Files[tokensPath]), dummy) {
		t.Error("tokens.gitconfig holds a token")
	}
}

func TestTokensFollowTheFileWhole(t *testing.T) {
	m, tk := tokenFixture(twoTokens)
	applyTokens(t, tk)
	m.Files[TokensFile] = []byte(twoTokens[:strings.Index(twoTokens, "\n\n")+1])
	applyTokens(t, tk)
	if cfg := string(m.Files[tokensPath]); strings.Contains(cfg, "b.example") || !strings.Contains(cfg, `[credential "https://a.example/x"]`) {
		t.Errorf("tokens.gitconfig = %q, want a.example/x's program alone", cfg)
	}
}

func TestTokensRemovesTheConfigWithoutTheFile(t *testing.T) {
	m, tk := tokenFixture(twoTokens)
	applyTokens(t, tk)
	delete(m.Files, TokensFile)
	if got := applyTokens(t, tk); len(got) != 1 {
		t.Errorf("plan = %q, want the removal", got)
	}
	if _, ok := m.Files[tokensPath]; ok {
		t.Error("tokens.gitconfig stays without the file")
	}
	if again := applyTokens(t, tk); again != nil {
		t.Errorf("second plan = %q, want none", again)
	}
}

func TestTokensWithoutTheFileChangeNothing(t *testing.T) {
	_, tk := tokenFixture("")
	if got := applyTokens(t, tk); got != nil {
		t.Errorf("plan = %q", got)
	}
}

func TestTokensRefuseBeforeWriting(t *testing.T) {
	m, tk := tokenFixture(`["https://a.example"]` + "\ntoken = \"has space\"\n")
	if _, err := tk.Plan(context.Background()); err == nil || strings.Contains(err.Error(), "has space") {
		t.Errorf("got %v", err)
	}
	if _, ok := m.Files[tokensPath]; ok {
		t.Error("tokens.gitconfig was written")
	}
}

func TestTokensConfigIsRootsWithTheAccountsGroup(t *testing.T) {
	m, tk := tokenFixture(twoTokens)
	applyTokens(t, tk)
	if got := m.Owners[tokensPath]; got != [2]int{0, 1000} {
		t.Errorf("owner = %v, want 0:1000", got)
	}
	if got := m.Modes[tokensPath]; got != 0o640 {
		t.Errorf("mode = %o, want 640", got)
	}
}
