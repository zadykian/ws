package wizard

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// WsSigning is the stage of ws's public signing key in ws server wizard. GitHub's API lists a
// login's signing keys with no token, so the stage checks them on ws. ws's gh has no account
// permission that may add the key, so where GitHub lacks it, the key goes to what stays by hand.
type WsSigning struct {
	Run  *WsRun
	M    effect.Machine
	Hand *Hand
	// named is set once this run named the key in what stays by hand, all that the stage can do.
	named bool
}

// Name is the stage's name.
func (*WsSigning) Name() string { return signingStage }

// WaitsOn is ws's run, which makes the key.
func (*WsSigning) WaitsOn() string { return runStage }

// Machine is ws: the stage reads its key and GitHub from there.
func (*WsSigning) Machine() Machine { return OnWs }

type signingByHand struct{ why string }

func (c signingByHand) String() string {
	return signingStage + ": would list ws's signing key in what stays by hand, as " + c.why
}

// Plan is empty where WS_USER's git config on ws names no key file, or GitHub lists the key among
// the signing keys of WS_USER's GitHub login.
func (s *WsSigning) Plan(ctx context.Context) (step.Plan, error) {
	line, err := signingLine(ctx, s.Run)
	if err != nil || line == "none" || s.named {
		return step.Plan{}, err
	}
	want, err := s.Run.Want()
	if err != nil {
		return step.Plan{}, fmt.Errorf("%s: %w", signingStage, err)
	}
	login := githubLoginOf(ctx, s.M, cmp.Or(want.User, s.Run.Default))
	if login == "" {
		return step.Plan{Changes: []step.Change{signingByHand{
			"neither WS_USER's gh nor its git config's github.user names a GitHub login"}}}, nil
	}
	keys, err := signingKeysOf(ctx, s.M, login)
	if err != nil || slices.Contains(keys, line) {
		return step.Plan{}, err
	}
	return step.Plan{Changes: []step.Change{signingByHand{"GitHub lists it for no login of WS_USER's"}}}, nil
}

// Apply lists the key in what stays by hand, to add on GitHub, as ws holds no token that may.
func (s *WsSigning) Apply(ctx context.Context, _ step.Plan) error {
	line, err := signingLine(ctx, s.Run)
	if err != nil {
		return err
	}
	s.Hand.Add("ws's signing key on GitHub: add it as a signing key at https://github.com/settings/ssh/new, "+
		"so that GitHub shows ws's commits as verified (docs/server-wizard.md): %s", line)
	s.Run.Out.Say("%s: GitHub lacks ws's signing key, which the list at the end names", signingStage)
	s.named = true
	return nil
}
