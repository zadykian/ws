package devtools

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
)

// TokenFile is the token that gh logs in with, on its last line, which mise writes from TokenVar.
const TokenFile = "/etc/ws/gh-token" //nolint:gosec // G101: a file's path, no credential

// TokenVar is the secret input that mise writes TokenFile from, and the variable that the secrets
// task's run sets to it.
const TokenVar = "WS_GITHUB_TOKEN" //nolint:gosec // G101: a variable's name, no credential

// RunToken returns the token that gh's login takes, empty where there is none. mise writes
// TokenFile in a run before any unit, but in a plan writes nothing. So where lookup gives the
// input, as in the secrets task's run, the token is its own, and a plan sees what the run writes.
// An empty one is no file. Without it, or with a nil lookup, TokenFile's last line gives it.
func RunToken(m effect.Machine, lookup func(string) (string, bool)) (string, error) {
	if lookup != nil {
		if text, ok := lookup(TokenVar); ok {
			return lastLine(text), nil
		}
	}
	text, err := m.ReadFile(TokenFile)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return lastLine(string(text)), err
}

// lastLine returns text's last line, as the file's tail -n 1 gives it, without its spaces.
func lastLine(text string) string {
	text = strings.TrimRight(text, "\n")
	return strings.TrimSpace(text[strings.LastIndex(text, "\n")+1:])
}

// GHLogin is gh's login to github.com by Token, the account's. Without a token, gh keeps its login.
type GHLogin struct {
	M     effect.Machine
	U     account.User
	Token string
	// Hidden names the secret inputs, which gh's env never holds, as the account's command reaches
	// no input of ws's run.
	Hidden []string
}

// gh returns the gh command of args as the account, from an env without Hidden.
func (g GHLogin) gh(args ...string) (string, []string) {
	argv := []string{}
	for _, name := range g.Hidden {
		argv = append(argv, "-u", name)
	}
	return "env", append(append(argv, "gh"), args...)
}

// Plan reads gh's token for github.com, and plans the login where that differs from Token. A gh
// with no login fails the read, which plans the login too.
func (g GHLogin) Plan(ctx context.Context) ([]resource.Change, error) {
	if g.Token == "" {
		return nil, nil
	}
	name, args := g.gh("auth", "token", "-h", "github.com")
	out, err := g.U.Output(ctx, g.M, name, args...)
	if err == nil && strings.TrimSpace(string(out)) == g.Token {
		return nil, nil
	}
	return []resource.Change{logIn{g}}, nil
}

// logIn passes the token to gh on its input, never in an argument, which a log or ps would show.
// ws has no keyring, so gh keeps it in ~/.config/gh/hosts.yml, mode 0600.
type logIn struct{ g GHLogin }

func (c logIn) String() string { return "log gh in to github.com by " + TokenVar }

func (c logIn) Apply(ctx context.Context) error {
	name, args := c.g.gh("auth", "login", "-h", "github.com", "-p", "https", "--insecure-storage", "--with-token")
	return c.g.U.PassInput(ctx, c.g.M, []byte(c.g.Token+"\n"), name, args...)
}
