package shell

import (
	"errors"
	"regexp"
)

// A URL names a section of tokens.gitconfig, which a quote or a backslash would rename or break. A
// word is what the helper hands git as a line: a token or a user name with no space in it. Python's
// \s is Unicode's, so these names the vertical tab and the spaces of every script too.
var (
	urlPattern  = regexp.MustCompile(`^https://[^\s\p{Z}\v"\\]+$`)
	wordPattern = regexp.MustCompile(`^[^\s\p{Z}\v]+$`)
)

// checkCredentials refuses the entries the role refused. The message never holds a token. keys are
// those of tokens.gitconfig, whose kept URLs take file names too.
func checkCredentials(creds []credential, keys []string) error {
	var bad bool
	files := map[string]bool{}
	for _, c := range creds {
		files[credentialFile(c.URL)] = true
		switch {
		case !urlPattern.MatchString(c.URL):
			bad = true
		case !wordPattern.MatchString(c.Token):
			bad = true
		case c.Username != "" && !wordPattern.MatchString(c.Username):
			bad = true
		}
	}
	kept := keptURLs(keys, creds)
	for _, u := range kept {
		files[credentialFile(u)] = true
	}
	if bad || len(files) != len(creds)+len(kept) {
		return errors.New("each [git] entry of the secrets file needs a URL of https:// without spaces, quotes or backslashes, " +
			"a token, and a username, if any, of one word each. " +
			"No two URLs, those of tokens.gitconfig included, may give the same file name: the URL without its scheme, " +
			"each run of other characters than letters, digits, ., _ and - as _")
	}
	return nil
}
