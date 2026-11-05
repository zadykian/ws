package wizard

import (
	"context"
	"errors"

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
)

// Sudo is the stage that finds how the remote scripts reach root on ws: a root login.
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

// Plan reads how a script reaches root. Another login stops the stage, until a later commit.
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
	return step.Plan{}, errors.New(sudoStage + ": " + w.User + "'s sudo on ws is not open to a script")
}
