package nvidia

import (
	"cmp"
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/apt"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

// loadedVersion is the version of the driver that the kernel runs.
const loadedVersion = "/sys/module/nvidia/version"

// reboot says a reboot is due while the kernel runs no driver, or another than the one installed,
// as the packages turn nouveau off and load theirs at boot. It plans no change.
type reboot struct {
	m   effect.Machine
	out effect.Out
}

func (r reboot) Plan(ctx context.Context) ([]resource.Change, error) {
	if !systemd.Booted(r.m) {
		r.out.Say("nvidia's driver: skipped, systemd is not running")
		return nil, nil
	}
	installed, err := installedVersion(ctx, r.m)
	if err != nil || installed == "" {
		return nil, err
	}
	loaded, err := r.m.ReadFile(loadedVersion)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if running := strings.TrimSpace(string(loaded)); running != installed {
		effect.SayLasting(r.out, "nvidia: the kernel runs NVIDIA's driver %s, and %s is installed. Reboot to load it: until then, ws-fans reads no GPU.",
			cmp.Or(running, "none"), installed)
	}
	return nil, nil
}

// installedVersion returns the upstream version of nvidia-smi's package, as the driver's module
// gives it, or "" where the package is not installed, as in a dry run.
func installedVersion(ctx context.Context, m effect.Machine) (string, error) {
	states, err := apt.PackageStates(ctx, m, smiPackage)
	if err != nil || !states[smiPackage].Installed() {
		return "", err
	}
	upstream, _, _ := strings.Cut(states[smiPackage].Version, "-")
	return upstream, nil
}
