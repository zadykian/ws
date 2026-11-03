package shell

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// envHeader is the top of ~/.secrets/env; ws replaces only a file that starts with its first
// words.
const (
	envMark   = "# Written by github.com/zadykian/ws"
	envHeader = envMark + " from shell_secret_env. Its next run with that variable replaces\n" +
		"# any change made here: set a value there.\n"
)

// envName is a name the shell takes as a variable.
var envName = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

// unsafeQuote matches a character that shlex.quote quotes.
var unsafeQuote = regexp.MustCompile(`[^A-Za-z0-9_@%+=:,./-]`)

// quote returns the string as Python's shlex.quote does: as it is where it holds only safe
// characters, else in single quotes, with each single quote as '"'"'.
func quote(s string) string {
	if s == "" {
		return "''"
	}
	if !unsafeQuote.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// checkEnvNames refuses the names the shell does not take, which go into the file as they are. The
// message names the names, never a value.
func checkEnvNames(env map[string]string) error {
	var bad []string
	for name := range env {
		if !envName.MatchString(name) {
			bad = append(bad, fmt.Sprintf("%q", name))
		}
	}
	slices.Sort(bad)
	if len(bad) > 0 {
		return fmt.Errorf("shell_secret_env names %s: a name takes capitals, digits and underscores, and starts with no digit", strings.Join(bad, ", "))
	}
	return nil
}

// renderEnv returns the content of ~/.secrets/env, a line per name in order.
func renderEnv(env map[string]string) []byte {
	var b strings.Builder
	b.WriteString(envHeader)
	names := make([]string, 0, len(env))
	for name := range env {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		b.WriteString("export " + name + "=" + quote(env[name]) + "\n")
	}
	return []byte(b.String())
}
