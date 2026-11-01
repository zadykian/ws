// Package fans is the fans role as Go steps: the NCT6687D's driver, nct6687, which DKMS builds
// for the board, and ws-fans, the service that runs ws server fans on the curves of
// /etc/ws-fans.json. See docs/fans.md.
package fans

import (
	"context"
	"embed"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

//go:embed files
var files embed.FS

// Step names. Those after the board's wait on the steps before them.
const (
	boardStep    = "Fans' board"
	packagesStep = "Fans' packages"
	sourceStep   = "Fans' driver source"
	driverStep   = "Fans' driver"
	modulesStep  = "Fans' modules"
	serviceStep  = "Fans' service"
)

// Steps returns the role's steps in order: the check of the board, the driver's packages and
// source, and the rest as they come. The role takes no secrets. On a machine with another board
// the steps after the first plan nothing.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	return steps(v.Fans, m, effect.OSUnits{}, out, running)
}

// steps is Steps with the fans' values, the units and the path of ws given, for a test to fake.
func steps(f values.Fans, m effect.Machine, u effect.Units, out effect.Out, exe func() (string, error)) []step.Step {
	vendor := f.BoardVendor
	return []step.Step{
		host.Group{Title: boardStep, Parts: []host.Resource{board{m, out, vendor}}},
		host.Lazy{Title: packagesStep, After: boardStep, Parts: gated(m, vendor, func(ctx context.Context) ([]host.Resource, error) {
			kernel, err := kernelRelease(ctx, m)
			if err != nil {
				return nil, err
			}
			return []host.Resource{host.Packages{M: m, Out: out, Recommends: true, Names: []string{"dkms", "linux-headers-generic", "linux-headers-" + kernel}}}, nil
		})},
		host.Lazy{Title: sourceStep, After: packagesStep, Parts: gated(m, vendor, func(context.Context) ([]host.Resource, error) {
			conf, err := host.Render(files, "files/dkms.conf.tmpl", struct{ Version string }{nct6687Short})
			if err != nil {
				return nil, err
			}
			return []host.Resource{
				source{m: m, sum: nct6687SHA256},
				host.File{M: m, Path: sourceDir + "/dkms.conf", Content: conf, Mode: 0o644},
			}, nil
		})},
		host.Lazy{Title: driverStep, After: sourceStep, Parts: gated(m, vendor, func(context.Context) ([]host.Resource, error) {
			return []host.Resource{driver{m}}, nil
		})},
		host.Lazy{Title: modulesStep, After: driverStep, Parts: gated(m, vendor, func(context.Context) ([]host.Resource, error) {
			return moduleParts(m, out)
		})},
		host.Lazy{Title: serviceStep, After: modulesStep, Parts: gated(m, vendor, serviceParts(m, u, out, f, exe))},
	}
}

// moduleParts keeps Linux's own driver off the chip, loads nct6687 at boot and now. The directory
// of the modules to load at boot is systemd's, which a machine without systemd may lack.
func moduleParts(m effect.Machine, out effect.Out) ([]host.Resource, error) {
	blacklist, err := files.ReadFile("files/modprobe.conf")
	if err != nil {
		return nil, err
	}
	load, err := files.ReadFile("files/modules-load.conf")
	if err != nil {
		return nil, err
	}
	return []host.Resource{
		host.File{M: m, Path: "/etc/modprobe.d/ws-fans.conf", Content: blacklist, Mode: 0o644},
		host.Dir{M: m, Path: "/etc/modules-load.d", Mode: 0o755},
		host.File{M: m, Path: "/etc/modules-load.d/ws-fans.conf", Content: load, Mode: 0o644},
		modules{m, out},
	}, nil
}

// kernelRelease returns the running kernel's release, as uname -r prints it.
func kernelRelease(ctx context.Context, m effect.Machine) (string, error) {
	out, err := m.Run(ctx, "uname", "-r")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
