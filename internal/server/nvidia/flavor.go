package nvidia

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"strings"

	"github.com/zadykian/ws/effect"
)

// flavorLine is the dpkg-query that lists the generic kernels' metapackages, their versions and
// their states.
var flavorLine = []string{"-W", "-f", "${Package} ${Version} ${db:Status-Abbrev}\n", "linux-image-generic*"}

// packageNames returns the driver's packages and the modules of each kernel flavor installed. The
// modules' metapackage of a flavor pulls the modules of each of its kernels, as the kernel moves.
// It goes at the version of the flavor's kernel metapackage: a newer one would pull a newer kernel
// without its headers, and DKMS would build no nct6687 for it. A kernel of none of the flavors
// fails the run, as no modules would follow it.
func packageNames(ctx context.Context, m effect.Machine) ([]string, error) {
	out, err := m.Run(ctx, "dpkg-query", flavorLine...)
	// dpkg-query exits with 1 where no package matches.
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return nil, err
	}
	names := slices.Clone(driverPackages)
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) != 3 || f[2] != "ii" {
			continue
		}
		if flavor, _ := strings.CutPrefix(f[0], "linux-image-"); slices.Contains(flavors, flavor) {
			names = append(names, modulesPrefix+flavor+"="+f[1])
		}
	}
	if len(names) == len(driverPackages) {
		return nil, errors.New("nvidia: no linux-image-generic of the flavors " + strings.Join(flavors, ", ") +
			" is installed, so no NVIDIA modules would follow the kernel")
	}
	return names, nil
}
