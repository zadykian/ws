package incus

import (
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
)

// Setup runs ws ubuntu guest setup in a guest, with the ws that the host pushed. Its plan is the
// guest's own, ws ubuntu guest setup --plan, so that a second run finds nothing due where the
// guest is in sync. A guest that does not run, or holds no ws, is due.
type Setup struct {
	M    effect.Machine
	Name string
	// Secrets runs the setup with --secrets, which clones the secrets on its first run.
	Secrets bool
	// Force plans the run whatever the guest's plan says, as ws ubuntu host guests setup asks.
	Force bool
	// Out shows what the guest's setup prints, which keeps no control character.
	Out effect.Out
}

// Plan reads the guest's plan once its boot is over.
func (s Setup) Plan(ctx context.Context) ([]resource.Change, error) {
	if s.Force {
		return []resource.Change{run{s}}, nil
	}
	up, err := running(ctx, s.M, s.Name)
	if err != nil || !up {
		return []resource.Change{run{s}}, err
	}
	if err := waitReady(ctx, s.M, s.Name); err != nil {
		return nil, err
	}
	args := []string{wsInGuest}
	if s.Secrets {
		args = append(args, "--secrets")
	}
	_, err = execIn(ctx, s.M, planTimeout, s.Name, planScript, args...)
	if code, ok := exitCode(err); ok && (code == 2 || code == missing) {
		return []resource.Change{run{s}}, nil
	}
	return nil, fromGuest(err)
}

type run struct{ s Setup }

func (c run) String() string {
	return fmt.Sprintf("run ws ubuntu guest setup in the guest %s", c.s.Name)
}

func (c run) Apply(ctx context.Context) error {
	if err := waitReady(ctx, c.s.M, c.s.Name); err != nil {
		return err
	}
	lines, ok := c.s.M.(effect.Lines)
	if !ok {
		return errors.New("this machine cannot filter what a guest prints")
	}
	args := []string{"exec", c.s.Name, "--", wsInGuest, "ubuntu", "guest", "setup"}
	if c.s.Secrets {
		args = append(args, "--secrets")
	}
	bounded, cancel := context.WithTimeout(ctx, setupTimeout)
	defer cancel()
	show := func(line string) bool {
		c.s.Out.Raw("%s", printable(line))
		return true
	}
	err := lines.PassLines(bounded, nil, show, "incus", args...)
	if err != nil && ctx.Err() == nil && bounded.Err() != nil {
		err = fmt.Errorf("the guest %s did not end its setup in %s, so ws stopped waiting: %w", c.s.Name, setupTimeout, err)
	}
	return fromGuest(err)
}
