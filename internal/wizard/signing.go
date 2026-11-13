package wizard

import (
	"context"
	"errors"
	guided "github.com/zadykian/ws/engine/core/wizard"

	"github.com/zadykian/ws/engine/core/step"
)

// signingStage names the stage in its messages.
const signingStage = "Signing key"

// Signing is the stage of ws's public signing key on GitHub, so that GitHub shows ws's commits as
// verified. The remote script's status prints that key. It runs after ws's run, which makes the key.
type Signing struct {
	Ws *Session
	// GitHub is where the stage looks for the key and what it does where GitHub lacks it: MacGitHub
	// or WsGitHub.
	GitHub SigningGitHub
}

// SigningGitHub is a Signing stage's side of GitHub.
type SigningGitHub interface {
	// lacks returns the change that adds line, where GitHub lacks it, or none.
	lacks(ctx context.Context, ws *Session, line string) (step.Change, error)
	// add adds line, or names it to add.
	add(ctx context.Context, ws *Session, line string) error
	// side is the side that the stage acts on.
	side() guided.Side
}

// Name is the stage's name.
func (*Signing) Name() string { return signingStage }

// Side is its GitHub's: the Mac, whose gh login takes ws's signing key, or ws.
func (s *Signing) Side() guided.Side { return s.GitHub.side() }

// WaitsOn is ws's run, which makes the key.
func (*Signing) WaitsOn() string { return runStage }

// line returns ws's signing key as status prints it, or none where WS_USER's git config on ws names
// no key file.
func (s *Signing) line(ctx context.Context) (string, error) { return signingLine(ctx, s.Ws) }

// signingLine is Signing's line, as ws's session reads it. It reads ws only where the remote script
// does.
func signingLine(ctx context.Context, ws *Session) (string, error) {
	s, ok := ws.status(ctx)
	if !ok {
		return "", errors.New(signingStage + ": the remote script reads nothing on ws, so the key cannot be read")
	}
	line := s.Signing
	if line == "" {
		return "", errors.New(signingStage + ": ws's status names no signing key")
	}
	return line, nil
}

// Plan is empty where WS_USER's git config on ws names no key file, or GitHub lists the key.
func (s *Signing) Plan(ctx context.Context) (step.Plan, error) {
	line, err := s.line(ctx)
	if err != nil || line == "none" {
		return step.Plan{}, err
	}
	c, err := s.GitHub.lacks(ctx, s.Ws, line)
	if err != nil || c == nil {
		return step.Plan{}, err
	}
	return step.Plan{Changes: []step.Change{c}}, nil
}

// Apply adds the key, or names it to add.
func (s *Signing) Apply(ctx context.Context, _ step.Plan) error {
	line, err := s.line(ctx)
	if err != nil {
		return err
	}
	return s.GitHub.add(ctx, s.Ws, line)
}
