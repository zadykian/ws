package shell

import (
	"context"
	"regexp"
	"slices"
	"strings"

	"github.com/zadykian/ws/effect"
)

// lines returns the non-empty lines of a command's output.
func lines(out []byte) []string {
	var l []string
	for _, line := range strings.Split(string(out), "\n") {
		if line != "" {
			l = append(l, line)
		}
	}
	return l
}

// subsection matches a key with a subsection: section.subsection.name.
var subsection = regexp.MustCompile(`^([^.]+)\..*\.([^.]+)$`)

// dropped returns the keys that a rewrite of ~/.gitconfig loses: those of the old file that the
// new one lacks. A subsection shows as '*', as one may be a URL, which names a host:
// http.*.extraheader stands for http.<url>.extraheader. The names alone, never a value.
func dropped(old, want []string) []string {
	var lost []string
	for _, k := range old {
		if !slices.Contains(want, k) {
			lost = append(lost, subsection.ReplaceAllString(k, "$1.*.$2"))
		}
	}
	slices.Sort(lost)
	return slices.Compact(lost)
}

// localSigning returns the signing settings among the keys of ~/.config/git/local, which
// ~/.gitconfig includes last, so that they win over the role's.
func localSigning(keys []string) []string {
	var found []string
	for _, k := range keys {
		if k == "user.signingkey" || k == "commit.gpgsign" || strings.HasPrefix(k, "gpg.") {
			found = append(found, k)
		}
	}
	slices.Sort(found)
	return slices.Compact(found)
}

// signingKeyOf returns the value of user.signingkey in the output of git config --list, as git
// reads a path: ~ stands for home. The last of several wins. The value is a path or a key's ID,
// not a secret.
func signingKeyOf(list []string, home string) (string, bool) {
	var key string
	var found bool
	for _, l := range list {
		if v, ok := strings.CutPrefix(l, "user.signingkey="); ok {
			key, found = v, true
		}
	}
	if v, ok := strings.CutPrefix(key, "~/"); ok {
		key = home + "/" + v
	}
	return key, found
}

// join lists names as prose does.
func join(names []string) string { return strings.Join(names, ", ") }

// gitNames lists the keys of a config's content, by git, without their values.
func gitNames(ctx context.Context, m effect.Machine, content []byte) ([]string, error) {
	out, err := m.RunInput(ctx, content, "git", "config", "--file", "-", "--name-only", "--list")
	return lines(out), err
}
