package setup

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// Cable gives the service that mac.conf's WS_CABLE_SERVICE names a Manual Configuration of the
// cable's address and mask with no router, through sudo, where networksetup -getinfo shows other
// settings. With no router, the cable takes no default route from the uplink. A service
// networksetup lacks stops the run, as does one with a router outside the cable's /30, as an
// uplink has. A disabled one draws a warning. Without WS_CABLE_SERVICE, it leaves the network alone.
type Cable struct {
	Conf conf.Conf
	M    effect.Machine
	Out  effect.Out
	// LookPath finds a program on the user's PATH. Nil means exec.LookPath.
	LookPath func(string) (string, error)
}

// Name is cable.
func (Cable) Name() string { return "cable" }

// Plan reads the service with networksetup.
func (c Cable) Plan(ctx context.Context) (step.Plan, error) {
	service := c.Conf.CableService
	if service == "" {
		return step.Plan{}, nil
	}
	look := c.LookPath
	if look == nil {
		look = exec.LookPath
	}
	// sudo gets the program's path as the user's PATH finds it, whatever PATH sudo would keep.
	program, err := look("networksetup")
	if err != nil {
		return step.Plan{}, fmt.Errorf("cannot find networksetup: put /usr/sbin, where macOS keeps it, on PATH")
	}
	list, err := c.M.RunShown(ctx, false, program, "-listallnetworkservices")
	if err != nil {
		return step.Plan{}, fmt.Errorf("networksetup -listallnetworkservices failed: %w", err)
	}
	listed, disabled := serviceListed(string(list), service)
	if !listed {
		return step.Plan{}, fmt.Errorf("networksetup -listallnetworkservices lacks %q, mac.conf's "+
			"WS_CABLE_SERVICE: name the display's adapter's service as the list prints it", service)
	}
	if disabled {
		c.Out.Warn("networksetup lists %q, mac.conf's WS_CABLE_SERVICE, as disabled, so the cable "+
			"carries nothing: turn it on in System Settings › Network", service)
	}
	info, err := c.M.RunShown(ctx, false, program, "-getinfo", service)
	if err != nil {
		return step.Plan{}, fmt.Errorf("networksetup -getinfo %q failed: %w", service, err)
	}
	other := cableOther(string(info))
	if other == "" {
		return step.Plan{}, nil
	}
	if router := uplinkRouter(string(info)); router != "" {
		return step.Plan{}, fmt.Errorf("networksetup -getinfo %q shows Router: %s, so the service routes "+
			"to another network: mac.conf's WS_CABLE_SERVICE must name the display's adapter's service",
			service, router)
	}
	return step.Plan{Changes: []step.Change{
		Run{
			Root: true, Name: program, Args: []string{"-setmanual", service, cableClient, cableMask},
			Label: fmt.Sprintf("networksetup -setmanual %q %s %s", service, cableClient, cableMask),
		},
	}}, nil
}
