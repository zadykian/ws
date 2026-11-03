package jetbrains

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/values"
)

// releasesURL is JetBrains' list of releases, which Gateway reads too: the latest release of each
// product, with a .tar.gz and its SHA-256 for each platform.
const releasesURL = "https://data.services.jetbrains.com/products/releases?code=%s&latest=true&type=release"

// timerRun is the value of WS_JETBRAINS_RUN that the timer's service sets.
const timerRun = "timer"

// backends makes the parts that install each backend's latest build, with Gateway's directory
// made first. The backends in place come first, as their removals need no download, so that each
// install finds the room that the older builds of those before it took.
func backends(v values.Values, site Site, run string) func(context.Context) ([]host.Resource, error) {
	return func(ctx context.Context) ([]host.Resource, error) {
		site.Home = v.Home
		site.Dist = filepath.Join(v.Home, ".cache/JetBrains/RemoteDev/dist")
		kind := "apply"
		if run == timerRun {
			kind = timerRun
		}
		site.Downloads = filepath.Join(v.Home, ".cache/ws-jetbrains", kind)
		// What a run cut short left goes, whatever its code, such as one since dropped.
		parts := []host.Resource{Stale{S: site, Path: site.Downloads}}
		if len(v.JetBrains.Backends) == 0 {
			return parts, nil
		}
		builds, err := site.latest(ctx, v.JetBrains.Backends)
		if err != nil {
			return nil, err
		}
		parts = append(parts, Dirs{S: site, Path: site.Dist})
		var inPlace, missing []host.Resource
		for _, b := range builds {
			bk := Backend{S: site, B: b}
			in, err := bk.installed()
			if err != nil {
				return nil, err
			}
			if in {
				inPlace = append(inPlace, bk)
			} else {
				missing = append(missing, bk)
			}
		}
		return append(append(parts, inPlace...), missing...), nil
	}
}

// platforms are the list's names for the architectures dpkg names.
var platforms = map[string]string{"amd64": "linux", "arm64": "linuxARM64"}

// latest reads the list of releases, and returns the latest build of each code.
func (s Site) latest(ctx context.Context, codes []string) ([]Build, error) {
	arch, err := s.M.Run(ctx, "dpkg", "--print-architecture")
	if err != nil {
		return nil, fmt.Errorf("reading dpkg's architecture: %w", err)
	}
	platform, ok := platforms[strings.TrimSpace(string(arch))]
	if !ok {
		platform = strings.TrimSpace(string(arch))
	}
	body, err := s.get(ctx, fmt.Sprintf(releasesURL, strings.Join(codes, ",")))
	if err != nil {
		return nil, err
	}
	return ParseReleases(body, codes, platform, s.Dist)
}
