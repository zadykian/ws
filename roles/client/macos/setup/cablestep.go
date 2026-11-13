package setup

import (
	"cmp"
	"context"
	"fmt"
	"os/exec"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
)

// Cable gives the service that mac.conf's WS_CABLE_SERVICE names a Manual Configuration of the
// cable's address and mask with no router. It does so through sudo, where networksetup -getinfo
// shows other settings. With no router, the cable takes no default route from the uplink. A service
// networksetup lacks stops the run, as does one with a router outside the cable's /30, as an uplink
// has. A disabled one draws a warning. Without WS_CABLE_SERVICE, it leaves the network alone.
type Cable struct {
	Conf conf.Conf
	M    effect.Machine
	Out  effect.Out
	// LookPath finds a program on the user's PATH. Nil means exec.LookPath.
	LookPath func(string) (string, error)
}

// Name is cable.
func (Cable) Name() string { return "cable" }

// WaitsOn is nothing.
func (Cable) WaitsOn() string { return "" }

// program finds networksetup. sudo gets its path as the user's PATH finds it, whatever PATH sudo
// would keep.
func (c Cable) program() (string, error) {
	look := c.LookPath
	if look == nil {
		look = exec.LookPath
	}
	program, err := look("networksetup")
	if err != nil {
		return "", fmt.Errorf("cannot find networksetup: put /usr/sbin, where macOS keeps it, on PATH")
	}
	return program, nil
}

// Report says how the service stands, where mac.conf names one. A disabled one is a conflict, as no
// run turns it on, and it keeps its address.
func (c Cable) Report(ctx context.Context) ([]step.Item, error) {
	if c.Conf.CableService == "" {
		return nil, nil
	}
	program, err := c.program()
	if err != nil {
		return nil, err
	}
	list, err := c.M.RunShown(ctx, false, program, "-listallnetworkservices")
	if err != nil {
		return nil, fmt.Errorf("networksetup -listallnetworkservices failed: %w", err)
	}
	info, err := c.M.RunShown(ctx, false, program, "-getinfo", c.Conf.CableService)
	if err != nil {
		return nil, fmt.Errorf("networksetup -getinfo %q failed: %w", c.Conf.CableService, err)
	}
	item := step.Item{Kind: step.ServiceItem, Name: c.Conf.CableService, Want: cableClient + " " + cableMask}
	item.Found = cmp.Or(cableOther(string(info)), item.Want)
	if _, disabled := serviceListed(string(list), c.Conf.CableService); disabled {
		item.Found += ", disabled"
		item.Conflict = "disabled, so the cable carries nothing: turn it on in System Settings › Network"
	}
	return []step.Item{item}, nil
}

// Plan reads the service with networksetup.
func (c Cable) Plan(ctx context.Context) (step.Plan, error) {
	service := c.Conf.CableService
	if service == "" {
		return step.Plan{}, nil
	}
	program, err := c.program()
	if err != nil {
		return step.Plan{}, err
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
		change.Run{
			Root: true, Name: program, Args: []string{"-setmanual", service, cableClient, cableMask},
			Label: fmt.Sprintf("networksetup -setmanual %q %s %s", service, cableClient, cableMask),
		},
	}}, nil
}

// Apply runs networksetup -setmanual, then reads the service again. Where it still shows other
// settings, as a profile that manages it would leave them, the run stops with the remedy. That is
// to set it by hand in System Settings › Network, as docs/mac.md's The cable says, or to empty
// mac.conf's WS_CABLE_SERVICE.
func (c Cable) Apply(ctx context.Context, p step.Plan) error {
	for _, ch := range p.Changes {
		if err := change.Apply(ctx, c.M, c.Out, ch); err != nil {
			return err
		}
		run, ok := ch.(change.Run)
		if !ok {
			continue
		}
		info, err := c.M.RunShown(ctx, false, run.Name, "-getinfo", c.Conf.CableService)
		if err != nil {
			return fmt.Errorf("networksetup -getinfo %q failed: %w", c.Conf.CableService, err)
		}
		if other := cableOther(string(info)); other != "" {
			return fmt.Errorf("networksetup -getinfo %q shows %s after %v: set it by hand in System "+
				"Settings › Network, as docs/mac.md's The cable says, or empty mac.conf's WS_CABLE_SERVICE",
				c.Conf.CableService, other, run.Line())
		}
		c.Out.Say("ran %s", run.Line())
	}
	return nil
}
