package secrets

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// stray is a host key that the secrets hold and that known_hosts gives no host, with the labels and
// the deploy keys' titles it sits under, sorted. A server that came back with another hostname and
// another key leaves its older ones so. Once ssh-keygen -R drops its line, nothing on the Mac ties
// them to the server.
type stray struct {
	Key   string
	Names []string
}

// strays are the ed25519 keys that a table of recipients of f or a deploy key of keys holds, and
// that known lacks, sorted by their first name. A recipient that is no host key, as the Nano's, is
// none.
func strays(f Files, keys []DeployKey, known map[string]bool) ([]stray, error) {
	names := map[string][]string{}
	note := func(key, name string) {
		if hostKeyForm.MatchString(key) && !known[key] && !slices.Contains(names[key], name) {
			names[key] = append(names[key], name)
		}
	}
	for _, file := range append([]string{SharedFile}, profileFiles(f)...) {
		text, ok := f[file]
		if !ok {
			continue
		}
		provider := ProfileProvider
		if file == SharedFile {
			provider = SharedProvider
		}
		r, err := Recipients(text, provider)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		for label, key := range r {
			note(key, label)
		}
	}
	for _, k := range keys {
		note(k.Key, k.Title)
	}
	strays := make([]stray, 0, len(names))
	for key, n := range names {
		slices.Sort(n)
		strays = append(strays, stray{Key: key, Names: n})
	}
	slices.SortFunc(strays, func(a, b stray) int {
		return cmp.Or(strings.Compare(a.Names[0], b.Names[0]), strings.Compare(a.Key, b.Key))
	})
	return strays, nil
}
