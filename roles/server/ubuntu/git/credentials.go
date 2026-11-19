package git

import (
	"embed"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/roles/internal/etcws"
	"github.com/zadykian/ws/roles/server/secrets"
)

// gitFiles is the template of the tokens' include.
//
//go:embed files/tokens.gitconfig.tmpl
var gitFiles embed.FS

// tokensDir holds each token in a file of its own, and tokensPath the credential programs that
// read them. Root owns both, with the account's group, so that the account reads a token and cannot
// change it.
const (
	tokensDir  = etcws.Dir + "/tokens"
	tokensPath = etcws.Dir + "/tokens.gitconfig"
)

// Credential is one server's token, from the [git] table of the secrets file.
type Credential struct {
	URL, Token, Username string
	ProactiveAuth        bool
}

// Credentials returns the table's entries, sorted by URL.
func Credentials(git map[string]secrets.Git) []Credential {
	var all []Credential
	for _, url := range slices.Sorted(maps.Keys(git)) {
		g := git[url]
		all = append(all, Credential{URL: url, Token: g.Token, Username: g.Username, ProactiveAuth: g.ProactiveAuth})
	}
	return all
}

// section is one URL's credential program in /etc/ws/tokens.gitconfig.
type section struct {
	URL, File string
	Proactive bool
}

// notFile matches each run of characters other than letters, digits, ., _ and -.
var notFile = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// credentialKey returns the key that names a URL to ws's credential program and in /etc/ws/tokens:
// the URL without its scheme, each run of other characters as _, so that a shell takes it as is.
func credentialKey(url string) string {
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
func keptURLs(keys []string, listed []Credential) []string {
	var kept []string
	for _, k := range keys {
		m := helperKey.FindStringSubmatch(k)
		if m == nil || slices.ContainsFunc(listed, func(c Credential) bool { return c.URL == m[1] }) {
			continue
		}
		kept = append(kept, m[1])
	}
	slices.Sort(kept)
	return slices.Compact(kept)
}

// sections returns the helpers of tokens.gitconfig, sorted by URL: the present entries' and the
// kept ones'. An entry's proactive auth is as the entry says, a kept one's as before.
func sections(keys []string, creds []Credential) []section {
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
		all = append(all, section{URL: u, File: credentialKey(u), Proactive: proactive[u]})
	}
	return all
}

// tokensConfig renders /etc/ws/tokens.gitconfig: the programs, which name the tokens' files
// and never a token.
func tokensConfig(s []section) ([]byte, error) {
	return resource.Render(gitFiles, "files/tokens.gitconfig.tmpl", struct{ Sections []section }{s})
}
