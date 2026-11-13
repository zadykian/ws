package wizard

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// githubLogin is a GitHub login, which goes into the API's address.
var githubLogin = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}$`)

// githubLoginOf returns user's GitHub login on ws: its gh's, else its git config's github.user, or an
// empty string. A login shell's profile may print first, so the last line counts.
func githubLoginOf(ctx context.Context, m effect.Machine, user string) string {
	out, err := m.Run(ctx, "runuser", "-l", user, "-c",
		"gh api user --jq .login 2>/dev/null || git config --global github.user")
	lines := strings.Fields(string(out))
	if err != nil || len(lines) == 0 || !githubLogin.MatchString(lines[len(lines)-1]) {
		return ""
	}
	return lines[len(lines)-1]
}

// signingKeysOf returns the type and base64 of each signing key that GitHub lists for login, from
// its public API, which takes no token.
func signingKeysOf(ctx context.Context, m effect.Machine, login string) ([]string, error) {
	out, err := m.Run(ctx, "curl", "-fsSL", "--retry", "3", "--proto", "=https",
		"https://api.github.com/users/"+login+"/ssh_signing_keys?per_page=100")
	var keys []struct{ Key string }
	if err != nil || json.Unmarshal(out, &keys) != nil {
		return nil, errors.New(SigningStage + ": curl read no list of signing keys from GitHub's API")
	}
	var lines []string
	for _, k := range keys {
		if f := strings.Fields(k.Key); len(f) > 1 {
			lines = append(lines, f[0]+" "+f[1])
		}
	}
	return lines, nil
}
