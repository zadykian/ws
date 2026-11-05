package tui

import (
	"context"
	"io"
	"syscall"

	tea "charm.land/bubbletea/v2"
)

// Config is what Run takes of the process.
type Config struct {
	// In gives the keys, and Out takes the screen and what stays by hand.
	In  io.Reader
	Out io.Writer
	// Theme is what the questions draw with.
	Theme Theme
	// Cut returns the error of a run that a signal ended. Ctrl-C counts as SIGINT. A command gives
	// the error that its plain run ends with, so that its exit status stays.
	Cut func(syscall.Signal) error
}

// Work is what a run does. It takes b as its effects, and sends the screen its messages by them.
type Work func(ctx context.Context, b *Bridge) error

// Run draws screen on c.Out, runs work in a goroutine beside it, and returns once the terminal is
// back. It returns the work's error, for execute to report. What stays by hand prints on the normal
// screen before Run returns.
func Run(ctx context.Context, c Config, screen tea.Model, work Work) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	bridge := newBridge(c.Out)
	begin := tea.Batch(
		func() tea.Msg { return doneMsg{work(ctx, bridge)} },
		func() tea.Msg { <-ctx.Done(); return endedMsg{} },
	)
	stop := func() tea.Msg { cancel(); return nil }
	model := session{screen: screen, theme: c.Theme, begin: begin, stop: stop, cut: c.Cut}
	program := tea.NewProgram(model, tea.WithInput(c.In), tea.WithOutput(c.Out))
	bridge.send = program.Send
	final, err := program.Run()
	close(bridge.gone)
	bridge.flush()
	if err != nil {
		return err
	}
	return final.(session).err //nolint:forcetypeassert // the program's model is a session
}
