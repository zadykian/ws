package git

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/internal/etcws"
)

// TokensFile is git's tokens for servers besides GitHub, a table per URL. mise writes it whole from
// the secret input WS_GIT_TOKENS of the role's mise.secrets.toml. Root owns it, with the
// account's group. So the account's git reads a token through ws's credential program, and cannot
// change it.
const TokensFile = etcws.Dir + "/git-tokens.toml"

// TokensVar is the secret input that mise writes TokensFile from, and the variable that the secrets
// task's run sets to it.
const TokensVar = "WS_GIT_TOKENS" //nolint:gosec // G101: a variable's name, no credential

// Token is one table of TokensFile, whose key is a server's URL, or a path on it.
type Token struct {
	Token string `toml:"token"`
	// Username is token where empty, as such servers take any.
	Username string `toml:"username"`
	// ProactiveAuth sends the token with git's first request, for a server that sends no 401.
	ProactiveAuth bool `toml:"proactive_auth"`
}

// ReadTokens reads TokensFile as the process's own user: root in the role's units, the account in
// git's credential program. present is false where the file is missing, as where the input is empty
// or the secrets task never ran. It refuses a key that ws does not know, and a table that
// checkCredentials refuses. No error holds a value.
func ReadTokens(m effect.Machine) (tokens map[string]Token, present bool, err error) {
	text, err := m.ReadFile(TokensFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	tokens, err = parseTokens(text, TokensFile)
	return tokens, err == nil, err
}

// RunTokens returns the tokens that the role's units act on. mise writes TokensFile from TokensVar
// in a run before any unit, but in a plan writes nothing. So where lookup gives the input, as in
// the secrets task's run, the tokens are its own, and a plan sees what the run writes. An empty one
// is no file. Without it, or with a nil lookup, they are ReadTokens'.
func RunTokens(m effect.Machine, lookup func(string) (string, bool)) (tokens map[string]Token, present bool, err error) {
	if lookup == nil {
		return ReadTokens(m)
	}
	text, ok := lookup(TokensVar)
	switch {
	case !ok:
		return ReadTokens(m)
	case text == "":
		return nil, false, nil
	}
	tokens, err = parseTokens([]byte(text), TokensVar)
	return tokens, err == nil, err
}

// parseTokens reads the TOML of text, which its error calls where. It refuses a key that ws does
// not know, and a table that checkCredentials refuses. No error holds a value.
func parseTokens(text []byte, where string) (tokens map[string]Token, err error) {
	d := toml.NewDecoder(bytes.NewReader(text))
	d.DisallowUnknownFields()
	if err := d.Decode(&tokens); err != nil {
		return nil, fmt.Errorf("%s: %w", where, scrub(err))
	}
	if err := checkCredentials(credentialsOf(tokens)); err != nil {
		return nil, fmt.Errorf("%s: %w", where, err)
	}
	return tokens, nil
}

// credentialsOf returns the file's tables, sorted by URL.
func credentialsOf(tokens map[string]Token) []Credential {
	var all []Credential
	for _, url := range slices.Sorted(maps.Keys(tokens)) {
		t := tokens[url]
		all = append(all, Credential{URL: url, Token: t.Token, Username: t.Username, ProactiveAuth: t.ProactiveAuth})
	}
	return all
}

// scrub keeps a value out of a decode error, which TOML words with the text it could not read. It
// gives the line alone, and the last part of a key that ws does not know, so not the URL either.
func scrub(err error) error {
	var strict *toml.StrictMissingError
	if errors.As(err, &strict) {
		var at []string
		for _, e := range strict.Errors {
			row, _ := e.Position()
			key := e.Key()
			at = append(at, fmt.Sprintf("line %d: %s is a key that ws does not know", row, key[len(key)-1]))
		}
		return errors.New(strings.Join(at, "; "))
	}
	var de *toml.DecodeError
	if errors.As(err, &de) {
		row, _ := de.Position()
		return fmt.Errorf("line %d: not valid TOML, or a value of the wrong type", row)
	}
	return errors.New("not valid TOML, or a value of the wrong type")
}
