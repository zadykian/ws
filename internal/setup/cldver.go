package setup

import (
	"regexp"
	"strconv"
	"strings"
)

var cldRelease = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

// cldVersion returns the release that the output of cld --version names, as cld 0.13.0 does, and
// whether it names one. A cld built from a clone has no release's version.
func cldVersion(out string) (string, bool) {
	v := strings.TrimPrefix(strings.TrimSpace(out), "cld ")
	return v, cldRelease.MatchString(v)
}

// cldOlder reports whether the release a comes before the release b. Both are of cldRelease's form.
func cldOlder(a, b string) bool {
	x, y := strings.Split(a, "."), strings.Split(b, ".")
	for i := range x {
		m, _ := strconv.Atoi(x[i])
		n, _ := strconv.Atoi(y[i])
		if m != n {
			return m < n
		}
	}
	return false
}
