package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/zadykian/ws/roles/internal/gitcfg"
)

// helperPattern matches the helpers of the credential sections, whose lines git config --get-regexp
// prints as the key, a space and the value. A section's name is its URL, empty for the bare
// [credential].
const helperPattern = `^credential\..*helper$`

// covers says whether a credential section for the URL from applies to the URL to. The bare
// section applies to every URL, and a URL to itself and to the paths below it.
func covers(from, to string) bool {
	from = strings.TrimSuffix(from, "/")
	return from == "" || to == from || strings.HasPrefix(to, from+"/")
}

// resetting returns the names of the credential sections in text whose empty helper resets the
// helpers of a token's URL. gh auth setup-git writes such sections with --global, below the
// include. The names are the sections', never a helper's value.
func resetting(text string, urls []string) []string {
	var names []string
	for line := range strings.Lines(text) {
		key, value, _ := strings.Cut(strings.TrimSuffix(line, "\n"), " ")
		if strings.TrimSpace(value) != "" {
			continue
		}
		section := strings.TrimSuffix(strings.TrimPrefix(key, "credential."), "helper")
		section = strings.TrimSuffix(section, ".")
		if !slices.ContainsFunc(urls, func(u string) bool { return covers(section, u) }) {
			continue
		}
		name := "[credential]"
		if section != "" {
			name = fmt.Sprintf("[credential %q]", section)
		}
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	return names
}

// credentials returns the note of the credential sections in ~/.gitconfig's text that reset a
// token's credential helper. They sit below the include, so that they win over tokens.gitconfig's. ws's file
// sets gh's program already, so that the sections can go.
func (s shadow) credentials(ctx context.Context, text []byte) ([]string, error) {
	var urls []string
	for _, c := range s.Creds {
		urls = append(urls, c.URL)
	}
	if len(urls) == 0 {
		return nil, nil
	}
	out, err := s.M.RunInput(ctx, text, "git", "config", "--file", "-", "--get-regexp", helperPattern)
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return nil, fmt.Errorf("git config cannot read ~/.gitconfig: %w", err)
	}
	if names := resetting(string(out), urls); len(names) > 0 {
		return []string{fmt.Sprintf("~/.gitconfig's credential sections %s, below the include, reset the helpers of %s "+
			"for the URLs under them: remove them, as %s sets gh's program already",
			strings.Join(names, ", "), tokensPath, gitcfg.Path)}, nil
	}
	return nil, nil
}
