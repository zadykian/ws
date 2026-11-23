package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// commandLineTools are Apple's command line tools, which git comes from. It plans nothing where
// xcode-select -p finds them.
type commandLineTools struct {
	M effect.Machine
	// Out says that setup waits, as the wait takes minutes.
	Out effect.Out
	// Poll is the wait between two looks for the tools.
	Poll time.Duration
}

type installTools struct{ t commandLineTools }

func (installTools) String() string { return "install the command line tools" }

func (installTools) Kind() step.ChangeKind { return step.Install }

// Plan finds the tools by xcode-select -p, which fails where there are none.
func (t commandLineTools) Plan(ctx context.Context) ([]resource.Change, error) {
	if _, err := t.M.Run(ctx, "xcode-select", "-p"); err == nil {
		return nil, nil
	}
	return []resource.Change{installTools{t}}, nil
}

// Apply asks for the install, which opens Apple's dialog, and waits until xcode-select -p finds the
// tools. A cancelled context ends the wait. Where the request fails, as one already open does, the
// wait goes on: the failure is part of the error where the wait ends unmet.
func (c installTools) Apply(ctx context.Context) error {
	t := c.t
	asked := t.M.Pass(ctx, false, "xcode-select", "--install")
	t.Out.Say("waiting for the command line tools: finish Apple's dialog, or press Ctrl-C to stop")
	for {
		if _, err := t.M.Run(ctx, "xcode-select", "-p"); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("stopped before the command line tools stood: %w", errors.Join(ctx.Err(), asked))
		case <-time.After(t.Poll):
		}
	}
}
