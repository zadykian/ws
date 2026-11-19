package git

import (
	"errors"
	"regexp"
)

// A URL names a section of tokens.gitconfig, which a quote or a backslash would rename or break. A
// word is what the program hands git as a line: a token or a user name with no space in it. Python's
// \s is Unicode's, so these names the vertical tab and the spaces of every script too.
var (
	urlPattern  = regexp.MustCompile(`^https://[^\s\p{Z}\v"\\]+$`)
	wordPattern = regexp.MustCompile(`^[^\s\p{Z}\v]+$`)
)

// checkCredentials refuses the tables the role refuses. The message never holds a token.
func checkCredentials(creds []Credential) error {
	var bad bool
	keys := map[string]bool{}
	for _, c := range creds {
		keys[credentialKey(c.URL)] = true
		switch {
		case !urlPattern.MatchString(c.URL):
			bad = true
		case !wordPattern.MatchString(c.Token):
			bad = true
		case c.Username != "" && !wordPattern.MatchString(c.Username):
			bad = true
		}
	}
	if bad || len(keys) != len(creds) {
		return errors.New("each table, from WS_GIT_TOKENS, needs a URL of https:// without spaces, quotes or backslashes, " +
			"a token, and a username, if any, of one word each. " +
			"No two URLs may give the same key: the URL without its scheme, " +
			"each run of other characters than letters, digits, ., _ and - as _")
	}
	return nil
}
