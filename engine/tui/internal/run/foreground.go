package run

import (
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/local"
)

// foreground runs a command that talks to the maintainer with the terminal, while the screen is
// suspended.
type foreground struct {
	b *Bridge
	m local.OSMachine
}

// Foreground returns the machine's PassForeground, behind the screen's suspension: the screen
// leaves the terminal first, and draws again once the command has ended. m is a machine that
// Machine returned, so that a root command gets its ticket, while the screen can still ask. Where
// the screen can't leave, the command doesn't run.
func (b *Bridge) Foreground(m local.OSMachine) effect.Foreground {
	return foreground{b, m}
}

// PassForeground suspends the screen, prints the directions a step gave for the command, runs it,
// and resumes the screen. The terminal comes back even after the command failed, and a screen that
// can't resume ends the run with its error.
// The ticket comes first, once: a question past the suspension would draw where nobody reads it.
func (f foreground) PassForeground(ctx context.Context, root bool, name string, args ...string) (err error) {
	m := f.m
	if root && m.Ticket != nil {
		if err := m.Ticket(); err != nil {
			return err
		}
	}
	m.Ticket = nil
	if f.b.suspend != nil {
		if err := f.b.suspend(); err != nil {
			return fmt.Errorf("leaving the screen for %s: %w", name, err)
		}
	}
	f.b.directions()
	defer func() {
		if f.b.resume == nil {
			return
		}
		if back := f.b.resume(); back != nil {
			err = errors.Join(err, fmt.Errorf("drawing the screen again after %s: %w", name, back))
		}
	}()
	return m.PassForeground(ctx, root, name, args...)
}
