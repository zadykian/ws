package host

import (
	"context"
	"errors"
	"fmt"
	"github.com/zadykian/ws/engine/core/remote"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

// sudoTries is how many passwords the stage takes, as sudo's own three.
const sudoTries = 3

// sudoWay is how a remote script reaches root on ws.
type sudoWay int

const (
	sudoUnread sudoWay = iota // the stage has not read it
	sudoLogin                 // ssh logs in as root
	sudoOpen                  // sudo needs no password, as for a cloud image's user
	sudoAsked                 // sudo takes the password that the stage asked
)

// Sudo is the stage that finds how the remote scripts reach root on ws: a root login, a sudo that
// needs no password, or the password, asked once a run. The password stays in this struct and in
// ssh's stdin, never in an argument, a message or the environment.
type Sudo struct {
	// SSH is the Mac's ssh to ws.
	SSH remote.SSH
	// Tty tells whether a terminal is there to ask. Check is set under --check.
	Tty, Check bool
	P          effect.Hidden
	Out        effect.Out

	way      sudoWay
	password string
}

// Name is the stage's name.
func (*Sudo) Name() string { return stage.Sudo }

// Side is ws: how its scripts reach root.
func (*Sudo) Side() guided.Side { return session.Ws }

// WaitsOn is the stage that checks the login, without which sudo answers nothing.
func (*Sudo) WaitsOn() string { return stage.Login }

// sudoAsk is the change that asks for the password.
type sudoAsk struct{ user string }

func (a sudoAsk) String() string {
	return stage.Sudo + ": would ask for " + a.user + "'s sudo password on ws, at the terminal"
}

// Plan reads how a script reaches root. Under --check at a terminal it asks too, as the Answers
// stage and ws client macos apply's sudo do. It asks since the stages after it read ws through sudo
// and change nothing. Elsewhere a password is Apply's to ask.
func (s *Sudo) Plan(ctx context.Context) (step.Plan, error) {
	if s.way != sudoUnread {
		return step.Plan{}, nil
	}
	w, err := s.SSH.Way(ctx)
	if err != nil {
		return step.Plan{}, errors.New(stage.Sudo + ": " + err.Error())
	}
	if w.User == "root" {
		s.way = sudoLogin
		return step.Plan{}, nil
	}
	_, err = s.SSH.Run(ctx, "-o", "BatchMode=yes", s.SSH.Host, "sudo", "-n", "true")
	switch {
	case err == nil:
		s.way = sudoOpen
		return step.Plan{}, nil
	case strings.Contains(err.Error(), "exit status 255"):
		return step.Plan{}, errors.New(stage.Sudo + ": ssh to ws failed: " + remote.Stderr(err))
	case s.Check && s.Tty:
		return step.Plan{}, s.ask(ctx, w.User)
	}
	return step.Plan{Changes: []step.Change{sudoAsk{w.User}}}, nil
}

// Apply asks for the password.
func (s *Sudo) Apply(ctx context.Context, p step.Plan) error {
	c, ok := p.Changes[0].(sudoAsk)
	if !ok {
		return fmt.Errorf("%s: unknown change %T", stage.Sudo, p.Changes[0])
	}
	return s.ask(ctx, c.user)
}

// ask takes the password, and has sudo check it with nothing behind it on stdin. With a script
// behind a wrong password, sudo would read the script's lines as its next two tries.
func (s *Sudo) ask(ctx context.Context, user string) error {
	if !s.Tty {
		return fmt.Errorf("%s: %s's sudo on ws asks for a password, and no terminal is here to ask it: "+
			"run the wizard at a terminal", stage.Sudo, user)
	}
	for try := 1; ; try++ {
		password, err := s.P.AskHidden(ctx, user+"'s sudo password on ws")
		if err != nil {
			return fmt.Errorf("%s: %w", stage.Sudo, err)
		}
		_, err = s.SSH.RunInput(ctx, []byte(password+"\n"), "-o", "BatchMode=yes", s.SSH.Host,
			"sudo", "-k", "-S", "-p", "''", "true")
		if err == nil {
			s.way, s.password = sudoAsked, password
			return nil
		}
		why := remote.Stderr(err)
		if !strings.Contains(why, "Authentication failed") {
			return fmt.Errorf("%s: sudo on ws stopped: %s", stage.Sudo, why)
		}
		if try == sudoTries {
			return fmt.Errorf("%s: sudo on ws refused the password %d times", stage.Sudo, sudoTries)
		}
		s.Out.Say("%s: sudo on ws refused the password", stage.Sudo)
	}
}

// Rooted returns the command and stdin that run cmd as root on ws, with in as cmd's stdin. -k drops
// a cached credential, so that the first line is always sudo's: with one, sudo would pass the line
// on. A nil Sudo runs cmd as the login, as the tests do.
func (s *Sudo) Rooted(cmd []string, in []byte) ([]string, []byte, error) {
	if s == nil {
		return cmd, in, nil
	}
	switch s.way {
	case sudoLogin:
		return cmd, in, nil
	case sudoOpen:
		return append([]string{"sudo", "-n"}, cmd...), in, nil
	case sudoAsked:
		return append([]string{"sudo", "-k", "-S", "-p", ""}, cmd...), append([]byte(s.password+"\n"), in...), nil
	}
	return nil, nil, errors.New(stage.Sudo + ": the stage has not read how to reach root on ws")
}
