package tui

import (
	"context"

	"github.com/zadykian/ws/effect"
)

// sudoQuestion names the machine, as the wizard asks for sudo's password on ws too.
const sudoQuestion = "sudo password on this machine"

// sudo gets sudo's ticket through a question, as no command can ask over a screen.
type sudo struct {
	m effect.Machine
	p effect.Hidden
}

// ticket returns once sudo holds a ticket. A ticket that is valid, or a sudo that wants no password,
// asks nothing. So does each root command past the first, and one past a ticket that has lapsed
// asks again, as sudo would. The password stays in this frame, and goes once sudo has checked it.
func (s *sudo) ticket(ctx context.Context) error {
	if _, err := s.m.Run(ctx, "sudo", "-n", "-v"); err == nil {
		return nil
	}
	password, err := s.p.AskHidden(ctx, sudoQuestion)
	if err != nil {
		return err
	}
	// Nothing follows the password: sudo would read the next line as its next try.
	_, err = s.m.RunInput(ctx, []byte(password+"\n"), "sudo", "-S", "-p", "", "-v")
	return err
}
