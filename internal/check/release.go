package check

import "regexp"

var (
	release     = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	releasesURL = regexp.MustCompile(`^(https://|http://127\.0\.0\.1:)[A-Za-z0-9._~:/-]+$`)
)

// Release reports whether s is the tag of a release: vX.Y.Z, with no leading zero. A pseudo-version,
// a version with +dirty and devel are builds of no release.
func Release(s string) bool { return release.MatchString(s) }

// ReleasesURL reports whether s is an address that ws's releases are fetched from. It starts with
// https://, or with http://127.0.0.1: as CI's stand-in has. It holds no blank, no quote and no
// other character that a shell reads.
func ReleasesURL(s string) bool { return releasesURL.MatchString(s) }
