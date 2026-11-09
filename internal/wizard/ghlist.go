package wizard

import (
	"context"
	"fmt"
	"strings"
)

// listed tells whether the keys of gh's login hold the key, and whether the login needs its scopes
// refreshed. A 404 from its keys or its signing keys shows it lacks one. gh's other errors stop it.
func (g *Gh) listed(ctx context.Context, gh, key string) (listed, refresh bool, err error) {
	keys, err := g.api(ctx, gh, "user/keys")
	if err != nil {
		return false, false, err
	}
	signs, err := g.api(ctx, gh, "user/ssh_signing_keys")
	if err != nil {
		return false, false, err
	}
	if keys == nil || signs == nil {
		return false, true, nil
	}
	for _, k := range keys {
		if k == key {
			return true, false, nil
		}
	}
	return false, false, nil
}

// api returns the keys that gh's login lists at path, or nil where the login lacks their scope.
func (g *Gh) api(ctx context.Context, gh, path string) ([]string, error) {
	out, err := g.M.Run(ctx, gh, "api", "--paginate", path, "--jq", ".[].key")
	if err != nil {
		if strings.Contains(err.Error(), "HTTP 404") {
			return nil, nil
		}
		return nil, fmt.Errorf("%s: gh api %s failed: %w", ghStage, path, err)
	}
	return append([]string{}, strings.Split(strings.TrimSpace(string(out)), "\n")...), nil
}
