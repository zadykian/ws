package devtools

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/zadykian/ws/effect"
)

// keyNoise are the armor's Comment and Version lines, which the keyserver swaps from one fetch to
// the next.
var keyNoise = regexp.MustCompile(`(?m)^(Comment|Version): .*\n`)

// fetchKey fetches the repository's key and checks that it holds its pinned certificate alone.
func fetchKey(ctx context.Context, m effect.Machine, repo Repository) ([]byte, error) {
	body, err := m.Run(ctx, "curl", "-fsSL", "--retry", "5", "--retry-delay", "10", "--retry-all-errors", repo.Key)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", repo.Key, err)
	}
	prints, err := fingerprints(string(body))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", repo.Key, err)
	}
	if !slices.Equal(prints, []string{repo.Fingerprint}) {
		return nil, fmt.Errorf("%s served the primary keys %s, not %s alone", repo.Key,
			strings.Join(prints, ", "), repo.Fingerprint)
	}
	return keyNoise.ReplaceAll(body, nil), nil
}
