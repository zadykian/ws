package wizard

import (
	"context"
	"errors"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// sudoStage names the stage in its messages.
const sudoStage = "Sudo"

// sudoWay is how a remote script reaches root on ws.
type sudoWay int

const (
	sudoUnread sudoWay = iota // the stage has not read it
	sudoLogin                 // ssh logs in as root
	sudoOpen                  // sudo needs no password, as for a cloud image's user
)

// Sudo is the stage that finds how the remote scripts reach root on ws: a root login, or a sudo that
// needs no password.
type Sudo struct {
	// Host is the Host key stage, whose ssh this one uses. Its Tty and Check are this stage's.
	Host *HostKey
	P    effect.Hidden

	way sudoWay
}

// Name is the stage's name.
func (*Sudo) Name() string { return sudoStage }

// WaitsOn is the stage that checks the login, without which sudo answers nothing.
func (*Sudo) WaitsOn() string { return rootStage }

// Plan reads how a script reaches root.
func (s *Sudo) Plan(ctx context.Context) (step.Plan, error) {
	if s.way != sudoUnread {
		return step.Plan{}, nil
	}
	w, err := s.Host.SSH.Way(ctx)
	if err != nil {
		return step.Plan{}, errors.New(sudoStage + ": " + err.Error())
	}
	if w.User == "root" {
		s.way = sudoLogin
		return step.Plan{}, nil
	}
	_, err = s.Host.SSH.Run(ctx, "-o", "BatchMode=yes", "ws", "sudo", "-n", "true")
	switch {
	case err == nil:
		s.way = sudoOpen
		return step.Plan{}, nil
	case strings.Contains(err.Error(), "exit status 255"):
		return step.Plan{}, errors.New(sudoStage + ": ssh to ws failed: " + stderrOf(err))
	}
	return step.Plan{}, errors.New(sudoStage + ": " + w.User + "'s sudo on ws asks for a password")
}
