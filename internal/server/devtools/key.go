package devtools

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	"github.com/zadykian/ws/effect"
)

// keyNoise are the armor's Comment and Version lines, which the keyserver swaps from one fetch to
// the next.
var keyNoise = regexp.MustCompile(`(?m)^(Comment|Version): .*\n`)

// fetchKey fetches the repository's key and checks that it holds a certificate at least, and
// pinned ones alone.
func fetchKey(ctx context.Context, m effect.Machine, repo Repository) ([]byte, error) {
	body, err := fetch(ctx, m, repo.Key)
	if err != nil {
		return nil, err
	}
	prints, err := fingerprints(string(body))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", repo.Key, err)
	}
	if len(prints) == 0 {
		return nil, fmt.Errorf("%s served no key", repo.Key)
	}
	for _, p := range prints {
		if !slices.Contains(repo.Fingerprints, p) {
			return nil, fmt.Errorf("%s served the primary key %s, which is not pinned", repo.Key, p)
		}
	}
	return keyNoise.ReplaceAll(body, nil), nil
}
