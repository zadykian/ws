package tui

import (
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/effect"
)

// foreground runs a command that talks to the maintainer with the terminal, while the screen is
// suspended.
type foreground struct {
	b *Bridge
	m effect.OSMachine
}

// Foreground returns the machine's PassForeground, behind the screen's suspension: the screen
// leaves the terminal first, and draws again once the command has ended. m is a machine that
// Machine returned, so that a root command gets its ticket. Where the screen can't leave, the
// command doesn't run.
func (b *Bridge) Foreground(m effect.OSMachine) effect.Foreground {
	return foreground{b, m}
}

// PassForeground suspends the screen, runs the command, and resumes the screen. The terminal comes
// back even after the command failed, and a screen that can't resume ends the run with its error.
func (f foreground) PassForeground(ctx context.Context, root bool, name string, args ...string) (err error) {
	if f.b.suspend != nil {
		if err := f.b.suspend(); err != nil {
			return fmt.Errorf("leaving the screen for %s: %w", name, err)
		}
	}
	defer func() {
		if f.b.resume == nil {
			return
		}
		if back := f.b.resume(); back != nil {
			err = errors.Join(err, fmt.Errorf("drawing the screen again after %s: %w", name, back))
		}
	}()
	return f.m.PassForeground(ctx, root, name, args...)
}
