package wizard

import (
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/step"
)

// signingStage names the stage in its messages.
const signingStage = "Signing key"

// Signing is the stage of ws's public signing key, which the remote script's status prints, among
// the signing keys of the Mac's gh login, so that GitHub shows ws's commits as verified. ws's own
// gh holds no token that may add it. It runs after ws's run, which makes the key, and after the
// gh stage, whose login lists and adds the key.
type Signing struct {
	Gh  *Gh
	Run *WsRun
	// Check is set under --check, where the gh stage is pending and has not made the login yet.
	Check bool
}

// Name is the stage's name.
func (*Signing) Name() string { return signingStage }

// WaitsOn is ws's run, which makes the key.
func (*Signing) WaitsOn() string { return runStage }

type signingAdd struct{ title string }

func (c signingAdd) String() string {
	return signingStage + ": would run gh ssh-key add --type signing --title " + c.title +
		" for ws's signing key"
}

// line returns ws's signing key as status prints it, or none where WS_USER's git config on ws names
// no key file. It reads ws only where the remote script does.
func (s *Signing) line(ctx context.Context) (string, error) {
	if !s.Run.Remote.Status(ctx) {
		return "", errors.New(signingStage + ": the remote script reads nothing on ws, so the key cannot be read")
	}
	line := s.Run.Remote.Get("signing")
	if line == "" {
		return "", errors.New(signingStage + ": ws's status names no signing key")
	}
	return line, nil
}

// Plan is empty where WS_USER's git config on ws names no key file, or the Mac's gh lists the key among
// its login's signing keys. gh's errors other than a 404, which shows its login lacks the scope,
// stop it, as the gh stage mends first.
func (s *Signing) Plan(ctx context.Context) (step.Plan, error) {
	line, err := s.line(ctx)
	if err != nil || line == "none" {
		return step.Plan{}, err
	}
	path, _, err := s.Gh.ghPath(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	if _, err := s.Gh.M.Run(ctx, path, "auth", "status", "-h", "github.com"); err != nil {
		if s.Check {
			s.Gh.Out.Say("%s: after %s on the Mac", signingStage, ghStage)
			return step.Plan{}, nil
		}
		return step.Plan{}, errors.New(signingStage + ": the Mac's gh has no login: the stage " + ghStage + " mends it")
	}
	keys, err := s.Gh.api(ctx, path, "user/ssh_signing_keys")
	if err != nil {
		return step.Plan{}, err
	}
	for _, k := range keys {
		if k == line {
			return step.Plan{}, nil
		}
	}
	return step.Plan{Changes: []step.Change{signingAdd{s.Run.Remote.Get("hostname")}}}, nil
}

// Apply adds the key to the Mac's gh login, titled with ws's hostname, from a file in the lock.
func (s *Signing) Apply(ctx context.Context, _ step.Plan) error {
	line, err := s.line(ctx)
	if err != nil {
		return err
	}
	path, _, err := s.Gh.ghPath(ctx)
	if err != nil {
		return err
	}
	title := s.Run.Remote.Get("hostname")
	file := s.Run.Host.Lock + "/signing.pub"
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
