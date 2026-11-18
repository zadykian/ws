package kind

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// ErrNoSecrets is the error of the secrets func on a machine with no secrets file. A secret file
// whose template calls it stays unchanged, so that a run without the file, as CI's, changes
// nothing.
var ErrNoSecrets = errors.New("this machine holds no secrets file")

// envName is a name that the shell takes as a variable.
var envName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// unsafeQuote matches a character that shlex.quote quotes.
var unsafeQuote = regexp.MustCompile(`[^A-Za-z0-9_@%+=:,./-]`)

// renderSecret fills a secret file's template as render does, with the func secrets too, which
// returns the side's secrets. Only a secret file's template may read them, as its content stays
// out of every diff.
func renderSecret(ctx context.Context, e Env, dir fs.FS, name string) ([]byte, error) {
	text, err := fs.ReadFile(dir, name)
	if err != nil {
		return nil, err
	}
	funcs := facts(ctx, e)
	funcs["secrets"] = func() (any, error) {
		if e.Secrets == nil {
			return nil, ErrNoSecrets
		}
		return e.Secrets(ctx)
	}
	return fill(name, text, funcs, e.Data)
}

// shellQuote returns the string as Python's shlex.quote does: unchanged where it holds only safe
// characters, else in single quotes, with each single quote as '"'"'.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !unsafeQuote.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// exports returns a line `export NAME=VALUE` for each name, in order, the value quoted for the
// shell. A name that the shell does not take fails it, and the error names the names, never a value.
func exports(env map[string]string) (string, error) {
	names := slices.Sorted(maps.Keys(env))
	var bad []string
	for _, name := range names {
		if !envName.MatchString(name) {
			bad = append(bad, fmt.Sprintf("%q", name))
		}
	}
	if len(bad) > 0 {
		return "", fmt.Errorf("env names %s: a name takes letters, digits and underscores, and starts with no digit",
			strings.Join(bad, ", "))
	}
	var b strings.Builder
	for _, name := range names {
		b.WriteString("export " + name + "=" + shellQuote(env[name]) + "\n")
	}
	return b.String(), nil
}
