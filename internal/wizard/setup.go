package wizard

import (
	"context"
	"errors"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"os/exec"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/stages/keys"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

// Setup is the stage that has ws client macos apply set up the Mac from mac.conf, after sudo's
// password rather than amid its lines. ws client macos apply changes only what differs, and is
// silent then, so a plan is empty once its own plan is.
type Setup struct {
	// Run plans the steps of ws client macos apply, or applies them where check is not set, and
	// reports whether any had changes to make.
	Run func(ctx context.Context, check bool) (bool, error)
	// Conf is the mac.conf that Run reads.
	Conf string
	// LookPath finds a program on PATH. Nil means exec.LookPath.
	LookPath func(string) (string, error)
	M        effect.Machine
	sudoed   bool
	// applied is set once ws client macos apply has run: the runner plans again after Apply. The plan
	// builds ws client macos apply's steps afresh, which forget what a step only warned of, as cld's
	// install.
	applied bool
}

// Name is the stage's name.
func (*Setup) Name() string { return stage.Setup }

// Side is the Mac: ws client macos apply.
func (*Setup) Side() guided.Side { return stage.Mac }

// WaitsOn is Homebrew, which ws client macos apply needs. Under --check, ws client macos apply's
// plan stands alone, as setup_after in sh did, and the git step says itself what waits on the
// login.
func (*Setup) WaitsOn() string { return stage.Homebrew }

// Plan runs ws client macos apply's plan, which prints each change it finds. Where PATH has no
// brew, ws client macos apply would stop, so the plan waits on Homebrew. It waits on the Mac key
// stage too, for a key to name.
func (s *Setup) Plan(ctx context.Context) (step.Plan, error) {
	if s.applied {
		return step.Plan{}, nil
	}
	look := s.LookPath
	if look == nil {
		look = exec.LookPath
	}
	if _, err := look("brew"); err != nil {
		return step.Plan{Changes: []step.Change{stage.Todo(stage.Setup + ": after Homebrew")}}, nil
	}
	if s.unnamed() {
		return step.Plan{Changes: []step.Change{stage.Todo(stage.Setup + ": after " + stage.Key)}}, nil
	}
	if !s.sudoed {
		switch err := effect.Validate(ctx, s.M); {
		case errors.Is(err, effect.ErrNoTicket):
			return step.Plan{}, err
		case err != nil:
			return step.Plan{}, errors.New("sudo -v failed; ws client macos apply needs sudo, as an admin of the Mac")
		}
		s.sudoed = true
	}
	changed, err := s.Run(ctx, true)
	if err != nil || !changed {
		return step.Plan{}, err
	}
	return step.Plan{Changes: []step.Change{stage.Todo(stage.Setup + ": would apply the steps above")}}, nil
}

// unnamed reports whether mac.conf lacks a key that the Mac key stage asks for. Under --check that
// stage writes no line, and ws client macos apply would stop for the key, or plan without the one
// that a run names. A file that cannot be read is Run's to report.
func (s *Setup) unnamed() bool {
	text, err := s.M.ReadFile(s.Conf)
	if err != nil {
		return false
	}
	c, errs := conf.Lines(string(text))
	return len(errs) == 0 && keys.Asked(c)
}

// Apply runs ws client macos apply. Where it stops, it has said why. Where it finishes, the stage
// is done, as a step that only warns, as cld's, leaves its change in the plan.
func (s *Setup) Apply(ctx context.Context, _ step.Plan) error {
	_, err := s.Run(ctx, false)
	s.applied = err == nil
	return err
}
