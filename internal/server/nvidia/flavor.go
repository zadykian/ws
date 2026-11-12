package nvidia

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// packageNames returns the driver's packages and the modules of each kernel flavor installed. The
// modules' metapackage of a flavor pulls the modules of each of its kernels, as the kernel moves.
// It goes at the version of the flavor's kernel metapackage: a newer one would pull a newer kernel
// without its headers, and DKMS would build no nct6687 for it. A kernel of none of the flavors
// fails the run, as no modules would follow it.
func packageNames(ctx context.Context, m effect.Machine) ([]string, error) {
	states, err := host.PackageStates(ctx, m, "linux-image-generic*")
	if err != nil {
		return nil, err
	}
	names := slices.Clone(driverPackages)
	// The keys are sorted, as dpkg lists the packages.
	for _, name := range slices.Sorted(maps.Keys(states)) {
		if !states[name].Installed() {
			continue
		}
		if flavor, _ := strings.CutPrefix(name, "linux-image-"); slices.Contains(flavors, flavor) {
			names = append(names, modulesPrefix+flavor+"="+states[name].Version)
		}
	}
	if len(names) == len(driverPackages) {
		return nil, errors.New("nvidia: no linux-image-generic of the flavors " + strings.Join(flavors, ", ") +
			" is installed, so no NVIDIA modules would follow the kernel")
	}
	return names, nil
}
