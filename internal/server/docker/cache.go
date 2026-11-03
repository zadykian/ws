package docker

import (
	"fmt"
	"regexp"
)

// Cache is the build cache's three limits, each in daemon.json's size format.
type Cache struct{ Reserved, MaxUsed, MinFree string }

// maxKeepDays is the longest age in days that fits five digits. Six may pass Go's longest
// duration, 106,751 days, which stops dockerd.
const maxKeepDays = 99999

// sizeFormat is what dockerd reads with go-units' RAMInBytes: a number, then b, or k, m, g, t or p
// with b, ib or nothing after it. Each letter takes either case, and 50Gi, with no b after its i,
// stops dockerd. dockerd --validate skips daemon.json's builder section, so the role checks it.
var sizeFormat = regexp.MustCompile(`^[0-9]+(\.[0-9]+)? ?([kKmMgGtTpP]([iI]?[bB])?|[bB])?$`)

// DefaultCache gives the limits as shares of a disk of size bytes: 10%, 80% and 20% of it, in whole
// GiB plus one, as BuildKit rounds them.
func DefaultCache(size uint64) Cache {
	gb := func(percent uint64) string { return fmt.Sprintf("%dGB", size*percent/100>>30+1) }
	return Cache{Reserved: gb(10), MaxUsed: gb(80), MinFree: gb(20)}
}

// Check refuses an age or a size that dockerd cannot parse. It would pass the validation of
// daemon.json, then stop dockerd as it restarts, and Docker would stay down until the file is
// fixed. A negative age would be read as 0, which leaves the age out.
func (c Cache) Check(keepDays int) error {
	if keepDays < 0 || keepDays > maxKeepDays {
		return fmt.Errorf("docker_build_cache_keep_days is %d: it takes a whole number of days up to %d, or 0 for no age limit",
			keepDays, maxKeepDays)
	}
	for _, s := range [][2]string{{"reserved_space", c.Reserved}, {"max_used_space", c.MaxUsed}, {"min_free_space", c.MinFree}} {
		if !sizeFormat.MatchString(s[1]) {
			return fmt.Errorf("docker_build_cache_%s is %q: it takes daemon.json's size format, as 50G, 50GB or 50GiB, but not 50Gi",
				s[0], s[1])
		}
	}
	return nil
}
