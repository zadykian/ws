package fans

import (
	"regexp"
	"slices"
	"strings"
)

// dkmsLine is a line of `dkms status nct6687d`: `nct6687d/VERSION, KERNEL, ARCH: installed`, or
// `nct6687d/VERSION: added`.
var dkmsLine = regexp.MustCompile(`^nct6687d/([^,:]+)[,:]`)

// dkmsState is what DKMS holds of the driver.
type dkmsState struct {
	// versions are the driver's versions in DKMS, each once, in the order listed.
	versions []string
	// installed is whether the pinned version is installed for the kernel.
	installed bool
}

// parseDKMS reads the output of `dkms status nct6687d` for the pinned version and the kernel.
func parseDKMS(out, version, kernel string) dkmsState {
	var s dkmsState
	prefix := "nct6687d/" + version + ", " + kernel + ", "
	for _, line := range strings.Split(out, "\n") {
		m := dkmsLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if !slices.Contains(s.versions, m[1]) {
			s.versions = append(s.versions, m[1])
		}
		if rest, ok := strings.CutPrefix(line, prefix); ok && strings.Contains(rest, ": installed") {
			s.installed = true
		}
	}
	return s
}
