package tui

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/zadykian/ws/effect"
)

const (
	// sudoQuestion names the machine, as the wizard asks for sudo's password on ws too.
	sudoQuestion = "sudo password on this machine"
	sudoRefused  = "sudo refused that password. "
	// sudoTries is the answers that sudo takes at a terminal.
	sudoTries = 3
	// sudoRenewal is well inside the 5 minutes that a ticket lasts by default.
	sudoRenewal = time.Minute
)

// sudo gets sudo's ticket through a question, as no command can ask over a screen.
type sudo struct {
	m effect.Machine
	p effect.Hidden
	// held says that the last look found a ticket.
	held atomic.Bool
}

// ticket returns once sudo holds a ticket. A ticket that is valid, or a sudo that wants no password,
// asks nothing. So does each root command past the first, and one past a ticket that has lapsed
// asks again, as sudo would. The password stays in this frame, and goes once sudo has checked it.
func (s *sudo) ticket(ctx context.Context) error {
	if _, err := s.m.Run(ctx, "sudo", "-n", "-v"); err == nil {
		s.held.Store(true)
		return nil
	}
	text := sudoQuestion
	var refused error
	for range sudoTries {
		password, err := s.p.AskHidden(ctx, text)
		if err != nil {
			return err
		}
		// Nothing follows the password: sudo would read the next line as its next try.
		if _, refused = s.m.RunInput(ctx, []byte(password+"\n"), "sudo", "-S", "-p", "", "-v"); refused == nil {
			return s.kept(ctx)
		}
		text = sudoRefused + sudoQuestion
	}
	return fmt.Errorf("sudo refused the password %d times: %w", sudoTries, refused)
}

// kept looks for the ticket that a right password leaves, as a root command would: sudo -S -v passes
// with none, as with timestamp_timeout=0, and each further question would end the same way.
func (s *sudo) kept(ctx context.Context) error {
	if _, err := s.m.Run(ctx, "sudo", "-n", "--", "true"); err != nil {
		return fmt.Errorf("%w: %w", effect.ErrNoTicket, err)
	}
	s.held.Store(true)
	return nil
}

// hold renews a ticket until ctx ends. A step can outlast one, and a child's own sudo, such as brew's,
// would then ask at the terminal under the screen. With no ticket it waits for one, as a failed look
// lands in the system's log, and a run may need no root at all.
func (s *sudo) hold(ctx context.Context) {
	ticker := time.NewTicker(sudoRenewal)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.held.Load() {
				_, err := s.m.Run(ctx, "sudo", "-n", "-v")
				s.held.Store(err == nil)
			}
		}
	}
}
