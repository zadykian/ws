package brew

import (
	"context"
	"errors"
	"os/exec"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// Cask is one Homebrew cask as a resource. It plans the install where none of the cask's files is
// there, and reports how the cask stands.
type Cask struct {
	M        effect.Machine
	Out      effect.Out
	LookPath func(string) (string, error)
	Failed   map[string]bool // the step's, so a later plan in the run skips a cask that failed
	Name     string
	There    bool   // whether its files are there
	Missing  string // why its files count as gone
	Holds    bool   // the app's cask: the changes after it wait on it
}

// Plan plans Homebrew's cask where none of its files is there. Where Homebrew has the cask but
// its files are gone, it warns instead: brew install would upgrade the cask. Without Homebrew it
// stops.
func (c Cask) Plan(ctx context.Context) ([]resource.Change, error) {
	changes, _, err := c.plan(ctx)
	return changes, err
}

// plan also says whether the files are there, or would be after the plan.
func (c Cask) plan(ctx context.Context) ([]resource.Change, bool, error) {
	if c.There {
		return nil, true, nil
	}
	if c.Failed[c.Name] {
		return nil, false, nil
	}
	look := c.LookPath
	if look == nil {
		look = exec.LookPath
	}
	if _, err := look("brew"); err != nil {
		return nil, false, errors.New("install Homebrew first: https://brew.sh")
	}
	if _, err := c.M.Run(ctx, "brew", "list", "--cask", c.Name); err != nil {
		return []resource.Change{Install{
			Command: resource.Command{M: c.M, Name: "brew", Args: []string{"install", "--cask", c.Name}, Of: step.Install},
			Out:     c.Out, Failed: c.Failed, Name: c.Name, Holds: c.Holds,
		}}, true, nil
	}
	c.Out.Warn("%s", CaskGone(c.Name, c.Missing))
	return nil, false, nil
}
