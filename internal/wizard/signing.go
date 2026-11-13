package wizard

import (
	"context"
	"errors"
	"fmt"

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
	side() Side
}

// Name is the stage's name.
func (*Signing) Name() string { return signingStage }

// WaitsOn is ws's run, which makes the key.
func (*Signing) WaitsOn() string { return runStage }

// MacGitHub is the Mac's gh login, which lists and adds the key. ws's own gh holds no token that may
// add it. The stage runs after the gh stage, which makes the login.
type MacGitHub struct {
	Gh GhCLI
	// Lock is the run's lock, which holds the key's file for gh.
	Lock string
	// Check is set under --check, where the gh stage is pending and has not made the login yet.
	Check bool
}

func (*MacGitHub) side() Side { return OnMac }

type signingAdd struct{ title string }

func (c signingAdd) String() string {
	return signingStage + ": would run gh ssh-key add --type signing --title " + c.title +
		" for ws's signing key"
}

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

// lacks is empty where the Mac's gh lists the key among its login's signing keys. gh's errors other
// than a 404, which shows its login lacks the scope, stop it, as the gh stage mends first.
func (s *MacGitHub) lacks(ctx context.Context, ws *Session, line string) (step.Change, error) {
	path, _, err := s.Gh.ghPath(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.Gh.M.Run(ctx, path, "auth", "status", "-h", "github.com"); err != nil {
		if s.Check {
			s.Gh.Out.Say("%s: after %s on the Mac", signingStage, ghStage)
			return nil, nil
		}
		return nil, errors.New(signingStage + ": the Mac's gh has no login: the stage " + ghStage + " mends it")
	}
	keys, err := s.Gh.api(ctx, path, "user/ssh_signing_keys")
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		if k == line {
			return nil, nil
		}
	}
	return signingAdd{ws.known().Hostname}, nil
}

// add adds the key to the Mac's gh login, titled with ws's hostname, from a file in the lock.
func (s *MacGitHub) add(ctx context.Context, ws *Session, line string) error {
	path, _, err := s.Gh.ghPath(ctx)
	if err != nil {
		return err
	}
	title := ws.known().Hostname
	file := s.Lock + "/signing.pub"
	if err := s.Gh.M.WriteFile(file, []byte(line+"\n"), 0o600); err != nil {
		return fmt.Errorf("%s: %w", signingStage, err)
	}
	add := "gh ssh-key add --type signing --title " + title
	if _, err := s.Gh.M.Run(ctx, path, "ssh-key", "add", file, "--type", "signing", "--title", title); err != nil {
		return fmt.Errorf("%s: %s failed", signingStage, add)
	}
	s.Gh.Out.Say("%s: ran %s for ws's signing key", signingStage, add)
	return nil
}
