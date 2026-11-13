package wizard

import (
	"context"
	"errors"
	"fmt"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/server/ubuntu/session"

	"github.com/zadykian/ws/engine/core/step"
)

// MacGitHub is the Mac's gh login, which lists and adds the key. ws's own gh holds no token that may
// add it. The stage runs after the gh stage, which makes the login.
type MacGitHub struct {
	Gh GhCLI
	// Lock is the run's lock, which holds the key's file for gh.
	Lock string
	// Check is set under --check, where the gh stage is pending and has not made the login yet.
	Check bool
}

func (*MacGitHub) Side() guided.Side { return OnMac }

type signingAdd struct{ title string }

func (c signingAdd) String() string {
	return SigningStage + ": would run gh ssh-key add --type signing --title " + c.title +
		" for ws's signing key"
}

// Lacks is empty where the Mac's gh lists the key among its login's signing keys. gh's errors other
// than a 404, which shows its login lacks the scope, stop it, as the gh stage mends first.
func (s *MacGitHub) Lacks(ctx context.Context, ws *session.Session, line string) (step.Change, error) {
	path, _, err := s.Gh.ghPath(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.Gh.M.Run(ctx, path, "auth", "status", "-h", "github.com"); err != nil {
		if s.Check {
			s.Gh.Out.Say("%s: after %s on the Mac", SigningStage, ghStage)
			return nil, nil
		}
		return nil, errors.New(SigningStage + ": the Mac's gh has no login: the stage " + ghStage + " mends it")
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
	return signingAdd{ws.Known().Hostname}, nil
}

// Add adds the key to the Mac's gh login, titled with ws's hostname, from a file in the lock.
func (s *MacGitHub) Add(ctx context.Context, ws *session.Session, line string) error {
	path, _, err := s.Gh.ghPath(ctx)
	if err != nil {
		return err
	}
	title := ws.Known().Hostname
	file := s.Lock + "/signing.pub"
	if err := s.Gh.M.WriteFile(file, []byte(line+"\n"), 0o600); err != nil {
		return fmt.Errorf("%s: %w", SigningStage, err)
	}
	add := "gh ssh-key add --type signing --title " + title
	if _, err := s.Gh.M.Run(ctx, path, "ssh-key", "add", file, "--type", "signing", "--title", title); err != nil {
		return fmt.Errorf("%s: %s failed", SigningStage, add)
	}
	s.Gh.Out.Say("%s: ran %s for ws's signing key", SigningStage, add)
	return nil
}
