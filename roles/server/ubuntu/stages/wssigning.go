package stages

import (
	"context"
	"fmt"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// WsGitHub is GitHub's API, from ws, for ws server ubuntu wizard. It lists a login's signing keys
// with no token, so the stage checks them on ws. ws's gh has no account permission that may add the
// key, so where GitHub lacks it, the key goes to what stays by hand.
type WsGitHub struct {
	M    effect.Machine
	Out  effect.Out
	Hand *guided.Hand
	// named is set once this run named the key in what stays by hand, all that the stage can do.
	named bool
}

// Side is ws: the stage acts through ws's own gh.
func (*WsGitHub) Side() guided.Side { return session.Ws }

type signingByHand struct{ why string }

func (c signingByHand) String() string {
	return SigningStage + ": would list ws's signing key in what stays by hand, as " + c.why
}

// Lacks is empty where GitHub lists the key among the signing keys of WS_USER's GitHub login.
func (s *WsGitHub) Lacks(ctx context.Context, ws *session.Session, line string) (step.Change, error) {
	if s.named {
		return nil, nil
	}
	user, err := ws.User()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", SigningStage, err)
	}
	login := githubLoginOf(ctx, s.M, user)
	if login == "" {
		return signingByHand{"neither WS_USER's gh nor its git config's github.user names a GitHub login"}, nil
	}
	keys, err := signingKeysOf(ctx, s.M, login)
	if err != nil || slices.Contains(keys, line) {
		return nil, err
	}
	return signingByHand{"GitHub lists it for no login of WS_USER's"}, nil
}

// Add lists the key in what stays by hand, to add on GitHub, as ws holds no token that may.
func (s *WsGitHub) Add(_ context.Context, _ *session.Session, line string) error {
	s.Hand.Add("ws's signing key on GitHub: add it as a signing key at https://github.com/settings/ssh/new, "+
		"so that GitHub shows ws's commits as verified (docs/server-wizard.md): %s", line)
	s.Out.Say("%s: GitHub lacks ws's signing key, which the list at the end names", SigningStage)
	s.named = true
	return nil
}
