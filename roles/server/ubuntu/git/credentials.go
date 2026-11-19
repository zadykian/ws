package git

import (
	"embed"
	"regexp"
	"strings"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/roles/internal/etcws"
)

// gitFiles is the template of the tokens' include.
//
//go:embed files/tokens.gitconfig.tmpl
var gitFiles embed.FS

// tokensPath holds the credential programs of TokensFile's URLs. Root owns it, with the account's
// group, as it names the servers.
const tokensPath = etcws.Dir + "/tokens.gitconfig"

// Credential is one server's token, a table of TokensFile.
type Credential struct {
	URL, Token, Username string
	ProactiveAuth        bool
}

// section is one URL's credential program in /etc/ws/tokens.gitconfig.
type section struct {
	URL, Key  string
	Proactive bool
}

// notKey matches each run of characters other than letters, digits, ., _ and -.
var notKey = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// credentialKey returns the key that names a URL to ws's credential program: the URL without its
// scheme, each run of other characters as _. A shell takes it as is.
func credentialKey(url string) string {
	return notKey.ReplaceAllString(strings.TrimPrefix(url, "https://"), "_")
}

// sections returns the sections of tokens.gitconfig, in the order of creds, which is the URLs'.
func sections(creds []Credential) []section {
	var all []section
	for _, c := range creds {
		all = append(all, section{URL: c.URL, Key: credentialKey(c.URL), Proactive: c.ProactiveAuth})
	}
	return all
}

// tokensConfig renders /etc/ws/tokens.gitconfig: the programs, which name the URLs' keys and never
// a token.
func tokensConfig(s []section) ([]byte, error) {
	return resource.Render(gitFiles, "files/tokens.gitconfig.tmpl", struct{ Sections []section }{s})
}
