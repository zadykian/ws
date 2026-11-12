package shell

import (
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

// credential is one server's token, from the [git] table of the secrets file.
type credential struct {
	URL, Token, Username string
	ProactiveAuth        bool
}

// credentials returns the table's entries, sorted by URL.
func credentials(git map[string]secrets.Git) []credential {
	var all []credential
	for _, url := range slices.Sorted(maps.Keys(git)) {
		g := git[url]
		all = append(all, credential{URL: url, Token: g.Token, Username: g.Username, ProactiveAuth: g.ProactiveAuth})
	}
	return all
}

// section is one URL's helper in ~/.config/git/tokens.gitconfig.
type section struct {
	URL, File string
	Proactive bool
}

// notFile matches each run of characters other than letters, digits, ., _ and -.
var notFile = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// credentialFile returns the name of a URL's token in ~/.config/git/tokens: the URL without its
// scheme, each run of other characters as _.
func credentialFile(url string) string {
	return notFile.ReplaceAllString(strings.TrimPrefix(url, "https://"), "_")
}

// helperKey matches the key of a URL's helper, and proactiveKey that of its proactive auth, in the
// output of git config --name-only --list.
var (
	helperKey    = regexp.MustCompile(`^credential\.(.+)\.helper$`)
	proactiveKey = regexp.MustCompile(`^http\.(.+)\.proactiveauth$`)
)

// keptURLs returns the URLs whose helpers tokens.gitconfig holds and the list omits: a run keeps
// their sections, as it keeps their tokens.
func keptURLs(keys []string, listed []credential) []string {
	var kept []string
	for _, k := range keys {
		m := helperKey.FindStringSubmatch(k)
		if m == nil || slices.ContainsFunc(listed, func(c credential) bool { return c.URL == m[1] }) {
			continue
		}
		kept = append(kept, m[1])
	}
	slices.Sort(kept)
	return slices.Compact(kept)
}

// sections returns the helpers of tokens.gitconfig, sorted by URL: the present entries' and the
// kept ones'. An entry's proactive auth is as the entry says, a kept one's as before.
func sections(keys []string, creds []credential) []section {
	kept := keptURLs(keys, creds)
	proactive := map[string]bool{}
	for _, k := range keys {
		if m := proactiveKey.FindStringSubmatch(k); m != nil && slices.Contains(kept, m[1]) {
			proactive[m[1]] = true
		}
	}
	urls := slices.Clone(kept)
	for _, c := range creds {
		urls = append(urls, c.URL)
		proactive[c.URL] = c.ProactiveAuth
	}
	slices.Sort(urls)
	var all []section
	for _, u := range slices.Compact(urls) {
		all = append(all, section{URL: u, File: credentialFile(u), Proactive: proactive[u]})
	}
	return all
}

// tokensConfig renders ~/.config/git/tokens.gitconfig: the helpers, which name the tokens' files
// and never a token.
func tokensConfig(s []section) ([]byte, error) {
	return host.Render(files, "files/tokens.gitconfig.tmpl", struct{ Sections []section }{s})
}
