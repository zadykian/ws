package devtools

import (
	"context"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// GHLogin logs gh in to github.com by Token, as User. ws has no keyring, so --insecure-storage
// puts the token in ~/.config/gh/hosts.yml, mode 0600. Without a Token the step plans nothing and
// leaves gh's login as it is.
type GHLogin struct {
	M     effect.Machine
	User  string
	Home  string
	Token string
}

type loginGH struct{ g GHLogin }

// String names the change, never the token.
func (loginGH) String() string { return "log gh in by token.github" }

// Apply gives the token to gh on its standard input. A refusal names the command, not the token.
func (c loginGH) Apply(ctx context.Context) error {
	u := host.User{Name: c.g.User, Home: c.g.Home}
	return u.PassInput(ctx, c.g.M, []byte(c.g.Token), "gh", "auth", "login", "-h", "github.com",
		"-p", "https", "--insecure-storage", "--with-token")
}

// Plan logs gh in where its token for github.com is not Token. A gh that holds none, or that is
// not installed yet, answers with an error, which counts as no login.
func (g GHLogin) Plan(ctx context.Context) ([]host.Change, error) {
	if g.Token == "" {
		return nil, nil
	}
	u := host.User{Name: g.User, Home: g.Home}
	if out, err := u.Output(ctx, g.M, "gh", "auth", "token", "-h", "github.com"); err == nil &&
		strings.TrimSpace(string(out)) == g.Token {
		return nil, nil
	}
	return []host.Change{loginGH{g}}, nil
}
