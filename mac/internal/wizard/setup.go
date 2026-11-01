package wizard

import (
	"context"
	"errors"
	"os/exec"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// setupStage names the stage in its messages.
const setupStage = "ws setup"

// Setup is the stage that has ws setup set up the Mac from mac.conf, after sudo's password rather
// than amid its lines. ws setup changes only what differs, and is silent then, so a plan is empty
// once its own plan is.
type Setup struct {
	// Run plans the steps of ws setup, or applies them where check is not set, and reports whether
	// any had changes to make.
	Run func(ctx context.Context, check bool) (bool, error)
	// LookPath finds a program on PATH. Nil means exec.LookPath.
	LookPath func(string) (string, error)
	M        effect.Machine
	sudoed   bool
	// applied is set once ws setup has run: the runner plans again after Apply, and the plan builds
	// ws setup's steps afresh, which forget what a step only warned of, as cld's install.
	applied bool
}

// Name is the stage's name.
func (*Setup) Name() string { return setupStage }

// WaitsOn is empty: under --check, ws setup's plan stands alone, as setup_after in sh did, and the
// git step says itself what waits on the login.
func (*Setup) WaitsOn() string { return "" }

// Plan runs ws setup's plan, which prints each change it finds. Where PATH has no brew, ws setup
// would stop, so the plan waits on Tools, bootstrap.sh's stage.
func (s *Setup) Plan(ctx context.Context) (step.Plan, error) {
	if s.applied {
		return step.Plan{}, nil
	}
	look := s.LookPath
	if look == nil {
		look = exec.LookPath
	}
	if _, err := look("brew"); err != nil {
		return step.Plan{Changes: []step.Change{todo(setupStage + ": after Tools")}}, nil
	}
	if !s.sudoed {
		if err := s.M.Pass(ctx, false, "sudo", "-v"); err != nil {
			return step.Plan{}, errors.New("sudo -v failed; ws setup needs sudo, as an admin of the Mac")
		}
		s.sudoed = true
	}
	changed, err := s.Run(ctx, true)
	if err != nil || !changed {
		return step.Plan{}, err
	}
	return step.Plan{Changes: []step.Change{todo(setupStage + ": would apply the steps above")}}, nil
}

// Apply runs ws setup. Where it stops, it has said why. Where it finishes, the stage is done, as a
// step that only warns, as cld's, leaves its change in the plan.
func (s *Setup) Apply(ctx context.Context, _ step.Plan) error {
	_, err := s.Run(ctx, false)
	s.applied = err == nil
	return err
}
