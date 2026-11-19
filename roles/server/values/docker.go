package values

import (
	"fmt"
	"regexp"
)

// maxKeepDays is the longest age in days that fits five digits. Six may pass Go's longest
// duration, 106,751 days, which stops dockerd.
const maxKeepDays = 99999

// sizeFormat is what dockerd reads with go-units' RAMInBytes: a number, then b, or k, m, g, t or p
// with b, ib or nothing after it. Each letter takes either case, and 50Gi, with no b after its i,
// stops dockerd. `dockerd --validate` skips daemon.json's builder section, so the values check it.
var sizeFormat = regexp.MustCompile(`^[0-9]+(\.[0-9]+)? ?([kKmMgGtTpP]([iI]?[bB])?|[bB])?$`)

// check refuses a build cache's age or size that dockerd cannot read. It would pass the validation
// of daemon.json, then stop dockerd as it restarts, and Docker would stay down until the file is
// fixed. A negative age would be read as 0, which leaves the age out. An empty size passes: the
// template fills a share of the disk.
func (d Docker) check() error {
	if d.BuildCacheKeepDays < 0 || d.BuildCacheKeepDays > maxKeepDays {
		return fmt.Errorf("WS_DOCKER_BUILD_CACHE_KEEP_DAYS is %d: it takes a whole number of days up to %d, or 0 for no age limit",
			d.BuildCacheKeepDays, maxKeepDays)
	}
	for _, s := range [...][2]string{
		{"WS_DOCKER_BUILD_CACHE_RESERVED_SPACE", d.BuildCacheReservedSpace},
		{"WS_DOCKER_BUILD_CACHE_MAX_USED_SPACE", d.BuildCacheMaxUsedSpace},
		{"WS_DOCKER_BUILD_CACHE_MIN_FREE_SPACE", d.BuildCacheMinFreeSpace},
	} {
		if s[1] != "" && !sizeFormat.MatchString(s[1]) {
			return fmt.Errorf("%s is %q: it takes daemon.json's size format, as 50G, 50GB or 50GiB, but not 50Gi", s[0], s[1])
		}
	}
	return nil
}
