package cable

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/engine/core/resource"
)

// setManual runs networksetup -setmanual, then reads the service again. Where it still shows other
// settings, as a profile that manages it would leave them, the run stops with the remedy. That is
// to set it by hand in System Settings › Network, as macos/README.md's The cable says, or to empty
// the server's WS_CABLE_SERVICE.
type setManual struct {
	c Cable
	resource.Command
}

// String says the command, which --check lists as due and a run says once it applied.
func (s setManual) String() string { return "run " + s.Line() }

// Apply runs the command, then checks the service.
func (s setManual) Apply(ctx context.Context) error {
	if err := s.Command.Apply(ctx); err != nil {
		return fmt.Errorf("%s failed: %w", s.Line(), err)
	}
	info, err := s.c.M.RunShown(ctx, false, s.Name, "-getinfo", s.c.Conf.CableService)
	if err != nil {
		return fmt.Errorf("networksetup -getinfo %q failed: %w", s.c.Conf.CableService, err)
	}
	if other := cableOther(string(info)); other != "" {
		return fmt.Errorf("networksetup -getinfo %q shows %s after %v: set it by hand in System "+
			"Settings › Network, as macos/README.md's The cable says, or empty WS_CABLE_SERVICE in the server's values.yml",
			s.c.Conf.CableService, other, s.Line())
	}
	return nil
}
