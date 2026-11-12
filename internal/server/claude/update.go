package claude

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
)

// releaseVersion is a version of a release, which cld update takes. A build from source, cld dev,
// it refuses.
var releaseVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

// Update updates cld where a newer release is out.
type Update struct {
	M    effect.Machine
	User host.User
	// Client and URL name the page that leads to the latest release. Zero values mean the
	// default client and cld's releases.
	Client *http.Client
	URL    string
}

type updateCld struct {
	u        Update
	have, to string
}

func (c updateCld) String() string { return fmt.Sprintf("update cld %s to %s", c.have, c.to) }

func (c updateCld) Apply(ctx context.Context) error {
	return c.u.User.Pass(ctx, c.u.M, c.u.User.Home+"/.local/bin/cld", "update")
}

// Plan compares cld's version with the release that releases/latest leads to, as cld update does.
// Without cld, the install step reports the change.
func (u Update) Plan(ctx context.Context) ([]host.Change, error) {
	cld := u.User.Home + "/.local/bin/cld"
	if _, err := u.M.Stat(cld); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	out, err := u.User.Output(ctx, u.M, cld, "--version")
	if err != nil {
		return nil, fmt.Errorf("cld --version: %w", err)
	}
	have := strings.TrimPrefix(strings.TrimSpace(string(out)), "cld ")
	if !releaseVersion.MatchString(have) {
		return nil, fmt.Errorf("cld %s is not a release, which cld update refuses", have)
	}
	url := u.URL
	if url == "" {
		url = cldReleases
	}
	latest, err := latestRelease(ctx, u.Client, url)
	if err != nil {
		return nil, err
	}
	if !semver.IsValid("v"+latest) || semver.Compare("v"+latest, "v"+have) <= 0 {
		return nil, nil
	}
	return []host.Change{updateCld{u, have, latest}}, nil
}
